package govarnam

import (
	"context"
	"fmt"
	"log"
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
		query = "INSERT OR IGNORE INTO words(word, confidence, learned_on) VALUES (trim(?), ?, strftime('%s', datetime(), 'localtime'))"
	}
	ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelfunc()
	stmt, err := varnam.dictConn.PrepareContext(ctx, query)
	checkError(err)
	defer stmt.Close()
	_, err = stmt.ExecContext(ctx, word, confidence)
	checkError(err)

	query = "UPDATE words SET confidence = confidence + 1 WHERE word = ?"
	ctx, cancelfunc = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelfunc()
	stmt, err = varnam.dictConn.PrepareContext(ctx, query)
	checkError(err)
	defer stmt.Close()
	_, err = stmt.ExecContext(ctx, word)
	checkError(err)
}

// Learn a word. If already exist, increases confidence of the pathway to that word.
// When learning a word, each path to that word is inserted into DB.
// Eg: ചങ്ങാതി: ചങ്ങ -> ചങ്ങാ -> ചങ്ങാതി
func (varnam *Varnam) Learn(word string) {
	conjuncts := varnam.splitWordByConjunct(strings.TrimSpace(word))

	if len(conjuncts) == 1 {
		varnam.insertWord(conjuncts[0], VARNAM_LEARNT_WORD_MIN_CONFIDENCE-1, false)
	} else {
		sequence := conjuncts[0]
		for i, ch := range conjuncts {
			if i == 0 {
				continue
			}
			sequence += ch
			if varnam.debug {
				fmt.Println(sequence)
			}
			if i+1 == len(conjuncts) {
				// The word. The final word should have the highest confidence
				varnam.insertWord(sequence, VARNAM_LEARNT_WORD_MIN_CONFIDENCE-1, false)
			} else {
				// Partial word. Part of pathway to the word to be learnt
				varnam.insertWord(sequence, VARNAM_LEARNT_WORD_MIN_CONFIDENCE-(len(conjuncts)-i), true)
			}
		}
	}
}

// Unlearn a word, remove from words DB and pattern if there is
func (varnam *Varnam) Unlearn(word string) {
	conjuncts := varnam.splitWordByConjunct(strings.TrimSpace(word))

	varnam.dictConn.Exec("PRAGMA foreign_keys = ON")

	for i := range conjuncts {
		// Loop will be going from full string to the first conjunct
		sequence := strings.Join(conjuncts[0:len(conjuncts)-i], "")
		if varnam.debug {
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
			_, err = stmt.ExecContext(ctx, word)
			checkError(err)

			// No need to remove from `patterns_content` since FOREIGN KEY ON DELETE CASCADE will work

			if varnam.debug {
				fmt.Printf("Removed %s\n", sequence)
			}
		}
	}

	varnam.dictConn.Exec("PRAGMA foreign_keys = OFF")
}

// Train a word with a particular pattern. Pattern => word
func (varnam *Varnam) Train(pattern string, word string) error {
	varnam.Learn(word)

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
