package govarnam

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"time"
)

type WordInfo struct {
	id         int
	word       string
	confidence int
	learnedOn  int
}

func checkError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

// Insert a word into word DB. Increment confidence if word exists
// Partial - Whether the word is not a real word, but only part of a pathway to a word
func (varnam *Varnam) insertWord(word string, confidence int, partial bool) {
	var query string

	if partial {
		query = "INSERT OR IGNORE INTO words(word, confidence, learned_on) VALUES (trim(?), ?, NULL)"
	} else {
		// The learned_on value determines whether it's a complete
		// word or just partial, i.e part of a path to a word
		query = "INSERT OR IGNORE INTO words(word, confidence, learned_on) VALUES (trim(?), ?, strftime('%s', 'now'))"
	}

	ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelfunc()
	stmt, err := varnam.dictConn.PrepareContext(ctx, query)
	checkError(err)
	defer stmt.Close()
	_, err = stmt.ExecContext(ctx, word, confidence)
	checkError(err)

	if partial {
		query = "UPDATE words SET confidence = confidence + 1 WHERE word = ?"
	} else {
		query = "UPDATE words SET confidence = confidence + 1, learned_on = strftime('%s', 'now') WHERE word = ?"
	}

	ctx, cancelfunc = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelfunc()
	stmt, err = varnam.dictConn.PrepareContext(ctx, query)
	checkError(err)
	defer stmt.Close()
	_, err = stmt.ExecContext(ctx, word)
	checkError(err)
}

// Sanitize a word, remove unwanted characters before learning
func sanitizeWord(word string) string {
	// Remove leading & trailing whitespaces
	word = strings.TrimSpace(word)

	// Remove leading ZWJ & ZWNJ
	firstChar, size := getFirstCharacter(word)
	if firstChar == ZWJ || firstChar == ZWNJ {
		word = word[size:len(word)]
	}

	// Remove trailing ZWJ & ZWNJ
	lastChar, size := getLastCharacter(word)
	if lastChar == ZWJ || lastChar == ZWNJ {
		word = word[0 : len(word)-size]
	}

	return word
}

// Learn a word. If already exist, increases confidence of the pathway to that word.
// When learning a word, each path to that word is inserted into DB.
// Eg: ചങ്ങാതി: ചങ്ങ -> ചങ്ങാ -> ചങ്ങാതി
func (varnam *Varnam) Learn(word string, confidence int) error {
	word = sanitizeWord(word)
	conjuncts, _ := varnam.splitWordByConjunct(word)

	if len(conjuncts) == 0 {
		return fmt.Errorf("Nothing to learn")
	} else if len(conjuncts) == 1 {
		// Forced learning of a single conjunct
		varnam.insertWord(conjuncts[0], VARNAM_LEARNT_WORD_MIN_CONFIDENCE-1, false)
	} else {
		sequence := conjuncts[0]
		for i, ch := range conjuncts {
			if i == 0 {
				continue
			}
			sequence += ch
			if varnam.Debug {
				fmt.Println("Learning", sequence)
			}
			if i+1 == len(conjuncts) {
				// The word. The final word should have the highest confidence

				var weight int

				// -1 because insertWord will increment one
				if confidence == 0 {
					weight = VARNAM_LEARNT_WORD_MIN_CONFIDENCE - 1
				} else {
					weight = confidence - 1
				}
				varnam.insertWord(sequence, weight, false)
			} else {
				// Partial word. Part of pathway to the word to be learnt
				varnam.insertWord(sequence, VARNAM_LEARNT_WORD_MIN_CONFIDENCE-(len(conjuncts)-i), true)
			}
		}
	}
	return nil
}

// Unlearn a word, remove from words DB and pattern if there is
func (varnam *Varnam) Unlearn(word string) error {
	conjuncts, _ := varnam.splitWordByConjunct(strings.TrimSpace(word))

	if len(conjuncts) == 0 {
		return fmt.Errorf("Nothing to unlearn")
	}

	varnam.dictConn.Exec("PRAGMA foreign_keys = ON")

	for i := range conjuncts {
		// Loop will be going from full string to the first conjunct
		sequence := strings.Join(conjuncts[0:len(conjuncts)-i], "")
		if varnam.Debug {
			fmt.Println(sequence)
		}

		// Check if there are words starting with this sequence
		rows, err := varnam.dictConn.Query("SELECT COUNT(*) FROM words WHERE word LIKE ?", sequence+"%")
		checkError(err)

		count := 0
		for rows.Next() {
			err := rows.Scan(&count)
			checkError(err)
		}

		if count == 1 {
			// If there's only one, remove it
			query := "DELETE FROM words WHERE word = ?"
			ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancelfunc()
			stmt, err := varnam.dictConn.PrepareContext(ctx, query)
			checkError(err)
			defer stmt.Close()
			_, err = stmt.ExecContext(ctx, sequence)
			checkError(err)

			// No need to remove from `patterns_content` since FOREIGN KEY ON DELETE CASCADE will work

			if varnam.Debug {
				fmt.Printf("Removed %s\n", sequence)
			}
		}
	}

	varnam.dictConn.Exec("PRAGMA foreign_keys = OFF")
	return nil
}

// Train a word with a particular pattern. Pattern => word
func (varnam *Varnam) Train(pattern string, word string) error {
	varnam.Learn(word, 0)

	wordInfo := varnam.getWordInfo(word)

	if wordInfo == nil {
		return fmt.Errorf("Word %s couldn't be inserted", word)
	}

	query := "INSERT OR IGNORE INTO patterns_content(pattern, word_id, learned) VALUES (?, ?, 1)"
	ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelfunc()
	stmt, err := varnam.dictConn.PrepareContext(ctx, query)
	checkError(err)
	defer stmt.Close()
	_, err = stmt.ExecContext(ctx, pattern, wordInfo.id)
	checkError(err)

	return nil
}

func (varnam *Varnam) getWordInfo(word string) *WordInfo {
	rows, err := varnam.dictConn.Query("SELECT id, confidence, learned_on FROM words WHERE word = ?", word)
	checkError(err)
	defer rows.Close()

	var wordInfo WordInfo
	wordExists := false

	for rows.Next() {
		// This loop will only work if there is a word
		wordExists = true

		rows.Scan(&wordInfo.id, &wordInfo.confidence, &wordInfo.learnedOn)
	}

	if wordExists {
		return &wordInfo
	}
	return nil
}

// LearnFromFile Learn all words in a file
func (varnam *Varnam) LearnFromFile(file io.Reader) {
	// io.Reader is a stream, so only one time iteration possible
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanWords)

	// First, see if this is a frequency report file
	// A frequency report file has the format :
	//    word frequency
	// Here the frequency will be the confidence
	frequencyReport := false

	word := ""
	count := 0
	for scanner.Scan() {
		curWord := scanner.Text()

		if count <= 1 {
			// Check the first 2 words. If it's of format <word frequency
			// Then treat rest of words as frequency report
			if count == 0 {
				word = curWord
				count++
				continue
			}

			confidence, err := strconv.Atoi(curWord)
			if err == nil {
				// It's a number
				frequencyReport = true
				varnam.Learn(word, confidence)
				word = ""
			} else {
				// Not a frequency report, so attempt to leatn those 2 words
				varnam.Learn(word, 0)
				varnam.Learn(curWord, 0)
			}

			if varnam.Debug {
				fmt.Println("Frequency report :", frequencyReport)
			}
		} else if frequencyReport {
			if word == "" {
				word = curWord
			} else {
				confidence, err := strconv.Atoi(curWord)

				if err == nil {
					varnam.Learn(word, confidence)
					count++
				}
				word = ""
			}
		} else {
			varnam.Learn(curWord, 0)
			count++
		}
		if count%500 == 0 {
			fmt.Printf("Learnt %d words\n", count)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}
