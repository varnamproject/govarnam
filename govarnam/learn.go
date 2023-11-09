package govarnam

/**
 * govarnam - An Indian language transliteration library
 * Copyright Subin Siby <mail at subinsb (.) com>, 2021
 * Licensed under AGPL-3.0-only. See LICENSE.txt
 */

import (
	"bufio"
	"context"
	sql "database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/mattn/go-sqlite3"
)

// WordInfo represent a item in words table
type WordInfo struct {
	id        int
	word      string
	weight    int
	learnedOn int
}

// LearnStatus output of bulk learn
type LearnStatus struct {
	TotalWords  int
	FailedWords int
}

// Learnings file export format
type exportFormat struct {
	WordsDict    []map[string]interface{} `json:"words"`
	PatternsDict []map[string]interface{} `json:"patterns"`
}

func (varnam *Varnam) languageSpecificSanitization(word string) string {
	if varnam.SchemeDetails.LangCode == "ml" {
		/* Malayalam has got two ways to write chil letters. Converting the old style to new atomic chil one */
		word = strings.Replace(word, "ന്‍", "ൻ", -1)
		word = strings.Replace(word, "ണ്‍", "ൺ", -1)
		word = strings.Replace(word, "ല്‍", "ൽ", -1)
		word = strings.Replace(word, "ള്‍", "ൾ", -1)
		word = strings.Replace(word, "ര്‍", "ർ", -1)
	}

	if varnam.SchemeDetails.LangCode == "hi" {
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
		word = word[size:]
	}

	// Remove trailing ZWNJ
	lastChar, size := getLastCharacter(word)
	if lastChar == ZWNJ {
		word = word[0 : len(word)-size]
	}

	return word
}

// Learn a word. If already exist, increases weight
func (varnam *Varnam) Learn(word string, weight int) error {
	word = varnam.sanitizeWord(word)
	conjuncts := varnam.splitWordByConjunct(word)

	if len(conjuncts) == 0 {
		return fmt.Errorf("Nothing to learn")
	}

	if len(conjuncts) == 1 {
		return fmt.Errorf("Can't learn a single conjunct")
	}

	// reconstruct word
	word = strings.Join(conjuncts, "")

	if weight == 0 {
		weight = VARNAM_LEARNT_WORD_MIN_WEIGHT - 1
	}

	query := "INSERT OR IGNORE INTO words(word, weight, learned_on) VALUES (trim(?), ?, strftime('%s', 'now'))"

	bgContext := context.Background()

	ctx, cancelFunc := context.WithTimeout(bgContext, 5*time.Second)
	defer cancelFunc()

	stmt, err := varnam.dictConn.PrepareContext(ctx, query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, word, weight)
	if err != nil {
		return err
	}

	query = "UPDATE words SET weight = weight + 1, learned_on = strftime('%s', 'now') WHERE word = ?"
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

// Unlearn a word, remove from words DB and pattern if there is
func (varnam *Varnam) Unlearn(word string) error {
	conjuncts := varnam.splitWordByConjunct(strings.TrimSpace(word))

	if len(conjuncts) == 0 {
		// Word must be english ? See if that's the case
		stmt, err := varnam.dictConn.Prepare("DELETE FROM patterns WHERE pattern = ?")
		if err != nil {
			return err
		}
		defer stmt.Close()

		result, err := stmt.Exec(word)
		if err != nil {
			return err
		}

		affected, err := result.RowsAffected()
		if err != nil {
			return err
		}

		if affected == 0 {
			return fmt.Errorf("nothing to unlearn")
		}
		return nil
	}

	varnam.dictConn.Exec("PRAGMA foreign_keys = ON")

	query := "DELETE FROM words WHERE word = ?"
	stmt, err := varnam.dictConn.Prepare(query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(word)
	if err != nil {
		return err
	}

	// No need to remove from `patterns` since FOREIGN KEY ON DELETE CASCADE will work

	if varnam.Debug {
		fmt.Printf("Removed %s\n", word)
	}

	varnam.dictConn.Exec("PRAGMA foreign_keys = OFF")
	return nil
}

// LearnMany words in bulk. Faster learning
func (varnam *Varnam) LearnMany(words []WordInfo) (LearnStatus, error) {
	var (
		insertionValues []string
		insertionArgs   []interface{}

		updationValues []string
		updationArgs   []interface{}

		learnStatus LearnStatus = LearnStatus{len(words), 0}
	)

	for _, wordInfo := range words {
		word := varnam.sanitizeWord(wordInfo.word)
		weight := wordInfo.weight
		conjuncts := varnam.splitWordByConjunct(word)

		if len(conjuncts) == 0 {
			log.Printf("Nothing to learn from %s", word)
			learnStatus.FailedWords++
			continue
		}

		if len(conjuncts) == 1 {
			log.Printf("Can't learn a single conjunct: %s", word)
			learnStatus.FailedWords++
			continue
		}

		// reconstruct word
		word = strings.Join(conjuncts, "")

		// We have a weight + 1 in SQL query later
		if weight == 0 {
			weight = VARNAM_LEARNT_WORD_MIN_WEIGHT - 1
		} else {
			weight--
		}

		insertionValues = append(insertionValues, "(trim(?), ?, strftime('%s', 'now'))")
		insertionArgs = append(insertionArgs, word, weight)

		updationValues = append(updationValues, "word = ?")
		updationArgs = append(updationArgs, word)
	}

	if len(insertionArgs) == 0 {
		return learnStatus, nil
	}

	query := fmt.Sprintf(
		"INSERT OR IGNORE INTO words(word, weight, learned_on) VALUES %s",
		strings.Join(insertionValues, ", "),
	)

	stmt, err := varnam.dictConn.Prepare(query)
	if err != nil {
		return learnStatus, err
	}

	_, err = stmt.Exec(insertionArgs...)
	if err != nil {
		return learnStatus, err
	}

	// There is a limit on number of OR that can be done
	// Reference: https://stackoverflow.com/questions/9570197/sqlite-expression-maximum-depth-limit
	depthLimit := sqlite3Conn.GetLimit(sqlite3.SQLITE_LIMIT_EXPR_DEPTH) - 1

	for len(updationValues) > 0 {
		lastIndex := int(math.Min(float64(depthLimit), float64(len(updationValues))))

		query = "UPDATE words SET weight = weight + 1, learned_on = strftime('%s', 'now') WHERE " + strings.Join(updationValues[0:lastIndex], " OR ")

		stmt, err = varnam.dictConn.Prepare(query)
		if err != nil {
			return learnStatus, err
		}
		defer stmt.Close()

		_, err = stmt.Exec(updationArgs[0:lastIndex]...)
		if err != nil {
			return learnStatus, err
		}

		updationValues = updationValues[lastIndex:]
		updationArgs = updationArgs[lastIndex:]
	}

	return learnStatus, nil
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

	query := "INSERT OR IGNORE INTO patterns(pattern, word_id) VALUES (?, ?)"
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
	rows, err := varnam.dictConn.Query("SELECT id, weight, learned_on FROM words WHERE word = ?", word)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var wordInfo WordInfo
	wordExists := false

	for rows.Next() {
		// This loop will only work if there is a word
		wordExists = true

		rows.Scan(&wordInfo.id, &wordInfo.weight, &wordInfo.learnedOn)
	}

	if wordExists {
		return &wordInfo, nil
	}
	return nil, fmt.Errorf("Word doesn't exist")
}

// LearnFromFile Learn all words in a file
func (varnam *Varnam) LearnFromFile(filePath string) (LearnStatus, error) {
	learnStatus := LearnStatus{0, 0}

	file, err := os.Open(filePath)
	if err != nil {
		return learnStatus, err
	}
	defer file.Close()

	limitVariableNumber := sqlite3Conn.GetLimit(sqlite3.SQLITE_LIMIT_VARIABLE_NUMBER)
	log.Printf("default SQLITE_LIMIT_VARIABLE_NUMBER: %d", limitVariableNumber)

	// We have 2 fields per item, word and weight
	insertsPerTransaction := int(float64(limitVariableNumber) / 2)

	// io.Reader is a stream, so only one time iteration possible
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanWords)

	// First, see if this is a frequency report file
	// A frequency report file has the format :
	//    word frequency
	// Here the frequency will be the weight
	frequencyReport := false

	fileFormatDetermined := false

	var words []WordInfo

	word := ""
	insertions := 0
	count := 0

	for scanner.Scan() {
		curWord := scanner.Text()

		if !fileFormatDetermined {
			// Check the first 2 words. If it's of format <word frequency>
			// Then treat rest of words as frequency report

			// Set the first word
			if count == 0 {
				word = curWord
				count++
				continue
			}

			// Then check the next word to see if it's a number
			weight, err := strconv.Atoi(curWord)
			if err == nil {
				// It's a number. It is a frequency report
				frequencyReport = true
				words = append(words, WordInfo{0, word, weight, 0})
				word = ""

				// count is now 1
			} else {
				// Second word is not a number but a string.
				// Not a frequency report, so attempt to learn those 2 words
				words = append(words, WordInfo{0, word, 0, 0})
				words = append(words, WordInfo{0, curWord, 0, 0})

				count++
			}

			fileFormatDetermined = true

			if varnam.Debug {
				fmt.Println("Frequency report :", frequencyReport)
			}
		} else if frequencyReport {
			number, numberErr := strconv.Atoi(curWord)
			if word == "" {
				// Sometimes word will be characters like <0xa0>
				// which won't be detected by Go's bufio.ScanWords
				if numberErr != nil {
					word = curWord
				}
				continue
			} else {
				if numberErr == nil {
					words = append(words, WordInfo{0, word, number, 0})
					count++
				}
				word = ""
			}
		} else {
			words = append(words, WordInfo{0, curWord, 0, 0})
			count++
		}

		if count == insertsPerTransaction {
			learnStatusBatch, err := varnam.LearnMany(words)

			if err != nil {
				return learnStatus, err
			}

			learnStatus.TotalWords += learnStatusBatch.TotalWords
			learnStatus.FailedWords += learnStatusBatch.FailedWords
			insertions += learnStatusBatch.TotalWords

			count = 0
			words = []WordInfo{}

			fmt.Printf("Processed %d words\n", insertions)
		}
	}

	if len(words) != 0 {
		learnStatusBatch, err := varnam.LearnMany(words)

		if err != nil {
			return learnStatus, err
		}

		learnStatus.TotalWords += learnStatusBatch.TotalWords
		learnStatus.FailedWords += learnStatusBatch.FailedWords

		insertions += len(words)
		fmt.Printf("Processed %d words\n", insertions)
	}

	if err := scanner.Err(); err != nil {
		return learnStatus, err
	}

	return learnStatus, nil
}

// TrainFromFile Train words with a particular pattern in bulk
func (varnam *Varnam) TrainFromFile(filePath string) (LearnStatus, error) {
	// The file should have the format :
	//    pattern word
	// The separation between pattern and word should just be a single whitespace

	learnStatus := LearnStatus{0, 0}

	file, err := os.Open(filePath)
	if err != nil {
		return learnStatus, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	lineCount := 0
	for scanner.Scan() {
		line := scanner.Text()
		wordsInLine := strings.Fields(line)

		if len(wordsInLine) == 2 {
			learnStatus.TotalWords++

			err := varnam.Train(wordsInLine[0], wordsInLine[1])
			if err != nil {
				learnStatus.FailedWords++
				fmt.Printf("Couldn't train %s => %s (%s) \n", wordsInLine[0], wordsInLine[1], err.Error())
			}
		} else if lineCount > 2 {
			fmt.Printf("Line %d is not in correct format \n", lineCount+1)
		}

		lineCount++
		if lineCount%500 == 0 {
			fmt.Printf("Trained %d words\n", lineCount)
		}
	}

	if err := scanner.Err(); err != nil {
		return learnStatus, err
	}
	return learnStatus, nil
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
func (varnam *Varnam) Export(filePath string, wordsPerFile int) error {
	if fileExists(filePath) {
		return fmt.Errorf("Output file already exists")
	}

	patternsCount := -1
	wordsCount := -1

	countRows, err := varnam.dictConn.Query("SELECT COUNT(*) AS patternsCount FROM patterns UNION SELECT COUNT(*) AS wordsCount FROM words")
	if err != nil {
		return err
	}
	defer countRows.Close()

	for countRows.Next() {
		if patternsCount == -1 {
			countRows.Scan(&patternsCount)
		} else {
			countRows.Scan(&wordsCount)
		}
	}

	totalPages := int(math.Ceil(float64(wordsCount) / float64(wordsPerFile)))

	if varnam.Debug {
		log.Printf("Words: %d. Patterns: %d", wordsCount, patternsCount)
		log.Printf("Pages: %d", totalPages)
	}

	page := 1
	for page <= totalPages {
		wordsTableQuery := fmt.Sprintf("SELECT word AS w, weight AS c, learned_on AS l FROM words ORDER BY c DESC LIMIT %d OFFSET %d", wordsPerFile, (page-1)*wordsPerFile)

		wordsRows, err := varnam.dictConn.Query(wordsTableQuery)
		if err != nil {
			return err
		}
		defer wordsRows.Close()

		wordsData, err := rowsToJSON(wordsRows)

		patternsRows, err := varnam.dictConn.Query(
			`
			SELECT pattern AS p, (
				SELECT word FROM words WHERE words.id = patterns.word_id
			) AS w
			FROM patterns
			WHERE patterns.word_id IN (
				SELECT id FROM words WHERE word IN (
					SELECT w FROM (
						` + wordsTableQuery + `
					)
				)
			)
			`,
		)
		if err != nil {
			return err
		}
		defer patternsRows.Close()

		patternsData, err := rowsToJSON(patternsRows)

		output := exportFormat{wordsData, patternsData}

		jsonData, err := json.Marshal(output)

		filePathWithPageNumber := filePath + "-" + fmt.Sprint(page) + ".vlf"
		err = os.WriteFile(filePathWithPageNumber, jsonData, 0644)
		if err != nil {
			return err
		}

		page++
	}

	return nil
}

// Import learnings from file
func (varnam *Varnam) Import(filePath string) error {
	if !fileExists(filePath) {
		return fmt.Errorf("Import file not found")
	}

	// TODO better reading of JSON. This loads entire file into memory
	fileContent, _ := os.ReadFile(filePath)

	var dbData exportFormat

	if err := json.Unmarshal(fileContent, &dbData); err != nil {
		return fmt.Errorf("Parsing JSON failed, err: %s", err.Error())
	}

	limitVariableNumber := sqlite3Conn.GetLimit(sqlite3.SQLITE_LIMIT_VARIABLE_NUMBER)
	log.Printf("default SQLITE_LIMIT_VARIABLE_NUMBER: %d", limitVariableNumber)

	insertsPerTransaction := int(math.Min(
		float64(limitVariableNumber)/4, // We have 4 fields per item
		float64(len(dbData.WordsDict)),
	))

	var (
		args   []interface{}
		values []string
	)

	insertions := 0
	count := 0
	for i, item := range dbData.WordsDict {
		values = append(values, "(trim(?), ?, ?)")
		args = append(args, item["w"], item["c"], item["l"])

		count++
		if count == insertsPerTransaction || i == len(dbData.WordsDict)-1 {
			query := fmt.Sprintf(
				"INSERT OR IGNORE INTO words(word, weight, learned_on) VALUES %s",
				strings.Join(values, ", "),
			)

			stmt, err := varnam.dictConn.Prepare(query)
			if err != nil {
				return err
			}

			_, err = stmt.Exec(args...)
			if err != nil {
				return err
			}

			args = nil
			values = nil

			insertions += count
			count = 0

			fmt.Printf("Inserted %d words\n", insertions)
		}
	}

	args = nil
	values = nil

	insertsPerTransaction = int(math.Min(
		float64(limitVariableNumber)/2, // We have 2 fields per item
		float64(len(dbData.PatternsDict)),
	))

	insertions = 0
	count = 0
	for i, item := range dbData.PatternsDict {
		values = append(values, "(?, (SELECT id FROM words WHERE word = ?))")
		args = append(args, item["p"], item["w"])

		count++
		if count == insertsPerTransaction || i == len(dbData.WordsDict)-1 {
			query := fmt.Sprintf(
				"INSERT OR IGNORE INTO patterns(pattern, word_id) VALUES %s",
				strings.Join(values, ", "),
			)

			stmt, err := varnam.dictConn.Prepare(query)
			if err != nil {
				return err
			}

			_, err = stmt.Exec(args...)
			if err != nil {
				return err
			}

			args = nil
			values = nil

			insertions += count
			count = 0

			fmt.Printf("Inserted %d patterns\n", insertions)
		}
	}

	return nil
}
