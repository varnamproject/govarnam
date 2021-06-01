package govarnam

import (
	"context"
	sql "database/sql"
	"log"
	"time"
)

// DictionaryResult result from dictionary search
type DictionaryResult struct {
	sugs                 []Suggestion
	exactMatch           bool
	longestMatchPosition int
}

func openDB(path string) *sql.DB {
	conn, err := sql.Open("sqlite3", path)
	if err != nil {
		log.Fatal(err)
	}
	return conn
}

func (varnam *Varnam) openDict(dictPath string) {
	varnam.dictConn = openDB(dictPath)
}

func makeDictionary(dictPath string) {
	// TODO pragmas include

	conn := openDB(dictPath)
	queries := [3]string{"CREATE TABLE IF NOT EXISTS metadata (key TEXT UNIQUE, value TEXT);",
		"CREATE TABLE IF NOT EXISTS words (id integer primary key, word text unique, confidence integer default 1, learned_on integer);",
		"CREATE TABLE IF NOT EXISTS patterns_content (pattern text, word_id integer, learned integer default 0, primary key(pattern, word_id)) WITHOUT rowid;"}

	for _, query := range queries {
		ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelfunc()
		stmt, err := conn.PrepareContext(ctx, query)
		checkError(err)
		defer stmt.Close()
		_, err = stmt.ExecContext(ctx)
		checkError(err)
	}

	defer conn.Close()
}

func (varnam *Varnam) searchDictionary(words []string, all bool) []Suggestion {
	likes := ""

	var vals []interface{}

	if all == true {
		// _% means a wildcard with a sequence of 1 or more
		// % means 0 or more and would include the word itself
		vals = append(vals, words[0]+"_%")
	} else {
		vals = append(vals, words[0])
	}

	for i, word := range words {
		if i == 0 {
			continue
		}
		likes += "OR word LIKE ? "
		if all == true {
			vals = append(vals, word+"_%")
		} else {
			vals = append(vals, word)
		}
	}

	rows, err := varnam.dictConn.Query("SELECT word, confidence FROM words WHERE word LIKE ? "+likes+" ORDER BY confidence DESC LIMIT 10", vals...)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	var results []Suggestion

	for rows.Next() {
		var item Suggestion
		rows.Scan(&item.Word, &item.Weight)
		results = append(results, item)
	}

	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}

	return results
}

func (varnam *Varnam) getFromDictionary(tokens []Token) DictionaryResult {
	// This is a temporary storage for tokenized words
	// Similar to usage in tokenizeWord
	var results []Suggestion

	foundPosition := 0
	var foundDictWords []Suggestion

	for i, t := range tokens {
		var tempFoundDictWords []Suggestion
		if t.tokenType == VARNAM_TOKEN_SYMBOL {
			if i == 0 {
				for _, possibility := range t.token {
					sug := Suggestion{possibility.value1, VARNAM_TOKEN_BASIC_WEIGHT - possibility.weight}
					results = append(results, sug)
					tempFoundDictWords = append(tempFoundDictWords, sug)
				}
			} else {
				for j, result := range results {
					till := result.Word
					tillWeight := result.Weight

					if tillWeight == -1 {
						continue
					}

					firstToken := t.token[0]
					results[j].Word += firstToken.value1
					results[j].Weight -= firstToken.weight

					search := []string{results[j].Word}
					searchResults := varnam.searchDictionary(search, false)

					if len(searchResults) > 0 {
						tempFoundDictWords = append(tempFoundDictWords, searchResults[0])
					} else {
						// No need of processing this anymore
						results[j].Weight = -1
					}

					for k, possibility := range t.token {
						if k == 0 {
							continue
						}

						newTill := till + possibility.value1

						search = []string{newTill}
						searchResults = varnam.searchDictionary(search, false)

						if len(searchResults) > 0 {
							tempFoundDictWords = append(tempFoundDictWords, searchResults[0])

							newWeight := tillWeight - possibility.weight

							sug := Suggestion{newTill, newWeight}
							results = append(results, sug)
						} else {
							result.Weight = -1
						}
					}
				}
			}
		}
		if len(tempFoundDictWords) > 0 {
			foundDictWords = tempFoundDictWords
			foundPosition = t.position
		}
	}

	return DictionaryResult{foundDictWords, foundPosition == tokens[len(tokens)-1].position, foundPosition}
}

func (varnam *Varnam) getMoreFromDictionary(words []Suggestion) [][]Suggestion {
	var results [][]Suggestion
	for _, sug := range words {
		search := []string{sug.Word}
		searchResults := varnam.searchDictionary(search, true)
		results = append(results, searchResults)
	}
	return results
}

func (varnam *Varnam) getFromPatternDictionary(pattern string) []Suggestion {
	rows, err := varnam.dictConn.Query("SELECT word, confidence FROM words WHERE id IN (SELECT word_id FROM patterns_content WHERE pattern LIKE ?) ORDER BY confidence DESC LIMIT 10", pattern+"%")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	var results []Suggestion

	for rows.Next() {
		var item Suggestion
		rows.Scan(&item.Word, &item.Weight)
		item.Weight += VARNAM_LEARNT_WORD_MIN_CONFIDENCE
		results = append(results, item)
	}

	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}

	return results
}
