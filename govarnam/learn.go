package govarnam

import (
	"bufio"
	"context"
	sql "database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
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

// Learnings file export format
type exportFormat struct {
	WordsDict    []map[string]interface{} `json:"words"`
	PatternsDict []map[string]interface{} `json:"patterns_content"`
}

// Insert a word into word DB. Increment confidence if word exists
// Partial - Whether the word is not a real word, but only part of a pathway to a word
func (varnam *Varnam) insertWord(word string, confidence int, partial bool) error {
	var query string

	if partial {
		query = "INSERT OR IGNORE INTO words(word, confidence, learned_on) VALUES (trim(?), ?, NULL)"
	} else {
		// The learned_on value determines whether it's a complete
		// word or just partial, i.e part of a path to a word
		query = "INSERT OR IGNORE INTO words(word, confidence, learned_on) VALUES (trim(?), ?, strftime('%s', 'now'))"
	}

	bgContext := context.Background()

	ctx, cancelFunc := context.WithTimeout(bgContext, 5*time.Second)
	defer cancelFunc()

	stmt, err := varnam.dictConn.PrepareContext(ctx, query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, word, confidence)
	if err != nil {
		return err
	}

	if partial {
		query = "UPDATE words SET confidence = confidence + 1 WHERE word = ?"
	} else {
		query = "UPDATE words SET confidence = confidence + 1, learned_on = strftime('%s', 'now') WHERE word = ?"
	}

	ctx, cancelFunc = context.WithTimeout(bgContext, 5*time.Second)
	defer cancelFunc()

	stmt, err = varnam.dictConn.PrepareContext(ctx, query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, word)
	if err != nil {
		return err
	}

	return nil
}

func (varnam *Varnam) languageSpecificSanitization(word string) string {
	if varnam.SchemeInfo.LangCode == "ml" {
		/* Malayalam has got two ways to write chil letters. Converting the old style to new atomic chil one */
		word = strings.Replace(word, "ന്‍", "ൻ", -1)
		word = strings.Replace(word, "ണ്‍", "ൺ", -1)
		word = strings.Replace(word, "ല്‍", "ൽ", -1)
		word = strings.Replace(word, "ള്‍", "ൾ", -1)
		word = strings.Replace(word, "ര്‍", "ർ", -1)
	}

	if varnam.SchemeInfo.LangCode == "hi" {
		/* Hindi's DANDA (Purna viram) */
		word = strings.Replace(word, "।", "", -1)
	}

	return word
}

// Sanitize a word, remove unwanted characters before learning
func (varnam *Varnam) sanitizeWord(word string) string {
	// Remove leading & trailing whitespaces
	word = strings.TrimSpace(word)

	word = varnam.languageSpecificSanitization(word)

	// Remove leading ZWJ & ZWNJ
	firstChar, size := getFirstCharacter(word)
	if firstChar == ZWJ || firstChar == ZWNJ {
		word = word[size:len(word)]
	}

	// Remove trailing ZWNJ
	lastChar, size := getLastCharacter(word)
	if lastChar == ZWNJ {
		word = word[0 : len(word)-size]
	}

	return word
}

// Learn a word. If already exist, increases confidence of the pathway to that word.
// When learning a word, each path to that word is inserted into DB.
// Eg: ചങ്ങാതി: ചങ്ങ -> ചങ്ങാ -> ചങ്ങാതി
func (varnam *Varnam) Learn(word string, confidence int) error {
	word = varnam.sanitizeWord(word)
	conjuncts, err := varnam.splitWordByConjunct(word)

	if err != nil {
		return err
	}

	if len(conjuncts) == 0 {
		return fmt.Errorf("Nothing to learn")
	}

	sequence := ""
	for i, ch := range conjuncts {
		sequence += ch
		if varnam.Debug {
			fmt.Println("Learning", sequence)
		}

		if i == len(conjuncts)-1 {
			// The word. The final word should have the highest confidence

			var weight int

			// -1 because insertWord will increment one
			if confidence == 0 {
				weight = VARNAM_LEARNT_WORD_MIN_CONFIDENCE - 1
			} else {
				weight = confidence - 1
			}

			err := varnam.insertWord(sequence, weight, false)
			if err != nil {
				return err
			}
		} else {
			// Partial word. Part of pathway to the word to be learnt
			varnam.insertWord(sequence, VARNAM_LEARNT_WORD_MIN_CONFIDENCE-(len(conjuncts)-i), true)
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
		if err != nil {
			return err
		}

		count := 0
		for rows.Next() {
			err := rows.Scan(&count)
			if err != nil {
				return err
			}
		}

		if count == 1 {
			// If there's only one, remove it
			ctx, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancelFunc()

			query := "DELETE FROM words WHERE word = ?"
			stmt, err := varnam.dictConn.PrepareContext(ctx, query)
			if err != nil {
				return err
			}
			defer stmt.Close()

			_, err = stmt.ExecContext(ctx, sequence)
			if err != nil {
				return err
			}

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
	word = varnam.sanitizeWord(word)

	err := varnam.Learn(word, 0)
	if err != nil {
		return err
	}

	wordInfo, err := varnam.getWordInfo(word)

	if wordInfo == nil {
		return fmt.Errorf("Word %s couldn't be inserted (%s)", word, err.Error())
	}

	ctx, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFunc()

	query := "INSERT OR IGNORE INTO patterns_content(pattern, word_id) VALUES (?, ?)"
	stmt, err := varnam.dictConn.PrepareContext(ctx, query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, pattern, wordInfo.id)
	if err != nil {
		return err
	}

	return nil
}

func (varnam *Varnam) getWordInfo(word string) (*WordInfo, error) {
	rows, err := varnam.dictConn.Query("SELECT id, confidence, learned_on FROM words WHERE word = ?", word)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var wordInfo WordInfo
	wordExists := false

	for rows.Next() {
		// This loop will only work if there is a word
		wordExists = true

		rows.Scan(&wordInfo.id, &wordInfo.confidence, &wordInfo.learnedOn)
	}

	if wordExists {
		return &wordInfo, nil
	}
	return nil, fmt.Errorf("Word doesn't exist")
}

// LearnFromFile Learn all words in a file
func (varnam *Varnam) LearnFromFile(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

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
			// Check the first 2 words. If it's of format <word frequency>
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
		return err
	}

	return nil
}

// TrainFromFile Train words with a particular pattern in bulk
func (varnam *Varnam) TrainFromFile(filePath string) error {
	// The file should have the format :
	//    pattern word
	// The separation between pattern and word should just be a single whitespace

	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	count := 0
	for scanner.Scan() {
		line := scanner.Text()
		wordsInLine := strings.Fields(line)

		if len(wordsInLine) == 2 {
			err := varnam.Train(wordsInLine[0], wordsInLine[1])
			if err != nil {
				fmt.Printf("Couldn't train %s => %s (%s) \n", wordsInLine[0], wordsInLine[1], err.Error())
			}
		} else if count > 2 {
			return fmt.Errorf("File format not correct")
		}

		count++
		if count%500 == 0 {
			fmt.Printf("Trained %d words\n", count)
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}

// Get full data from DB
func rowsToJSON(rows *sql.Rows) ([]map[string]interface{}, error) {
	// Dumping rows from SQL to JSON
	// Thanks lucidquiet https://stackoverflow.com/a/17885636/1372424
	// Thanks turkenh https://stackoverflow.com/a/29164115/1372424
	// Licensed CC-BY-SA 4.0

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	count := len(columns)
	tableData := make([]map[string]interface{}, 0)
	values := make([]interface{}, count)
	valuePtrs := make([]interface{}, count)

	for rows.Next() {
		for i := 0; i < count; i++ {
			valuePtrs[i] = &values[i]
		}
		rows.Scan(valuePtrs...)
		entry := make(map[string]interface{})
		for i, col := range columns {
			var v interface{}
			val := values[i]
			b, ok := val.([]byte)
			if ok {
				v = string(b)
			} else {
				v = val
			}
			entry[col] = v
		}
		tableData = append(tableData, entry)
	}

	return tableData, nil
}

// Export learnings as JSON to a file
func (varnam *Varnam) Export(filePath string) error {
	if fileExists(filePath) {
		return fmt.Errorf("output file already exists")
	}

	wordsRows, err := varnam.dictConn.Query("SELECT * FROM words")
	if err != nil {
		return err
	}
	defer wordsRows.Close()

	wordsData, err := rowsToJSON(wordsRows)

	patternsRows, err := varnam.dictConn.Query("SELECT * FROM patterns_content")
	if err != nil {
		return err
	}
	defer wordsRows.Close()

	patternsData, err := rowsToJSON(patternsRows)

	output := exportFormat{wordsData, patternsData}

	jsonData, err := json.Marshal(output)

	err = ioutil.WriteFile(filePath, jsonData, 0644)
	if err != nil {
		return err
	}

	return nil
}

// Import learnings from file
func (varnam *Varnam) Import(filePath string) error {
	fileContent, _ := ioutil.ReadFile(filePath)

	var dbData exportFormat

	if err := json.Unmarshal(fileContent, &dbData); err != nil {
		return fmt.Errorf("Parsing packs JSON failed, err: %s", err.Error())
	}

	count := 0
	for _, item := range dbData.WordsDict {
		stmt, err := varnam.dictConn.Prepare("INSERT OR IGNORE INTO words(id, word, confidence, learned_on) VALUES (?, trim(?), ?, ?)")
		if err != nil {
			return err
		}

		_, err = stmt.Exec(item["id"], item["word"], item["confidence"], item["learned_on"])
		if err != nil {
			return err
		}

		count++
		if count%500 == 0 {
			fmt.Printf("Inserted %d words\n", count)
		}
	}

	count = 0
	for _, item := range dbData.PatternsDict {
		stmt, err := varnam.dictConn.Prepare("INSERT OR IGNORE INTO patterns_content(pattern, word_id) VALUES (?, ?)")
		if err != nil {
			return err
		}

		_, err = stmt.Exec(item["pattern"], item["word_id"])
		if err != nil {
			return err
		}

		count++
		if count%500 == 0 {
			fmt.Printf("Inserted %d patterns\n", count)
		}
	}

	return nil
}
