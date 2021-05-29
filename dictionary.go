package main

import (
	sql "database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

var dictConn *sql.DB

// DictionaryResult result from dictionary search
type DictionaryResult struct {
	sugs                 []Suggestion
	exactMatch           bool
	longestMatchPosition int
}

func openDict() {
	var err error
	dictConn, err = sql.Open("sqlite3", "./ml.vst.learnings")
	if err != nil {
		log.Fatal(err)
	}
}

func searchDictionary(words []string, all bool) []Suggestion {
	likes := ""

	var vals []interface{}

	if all == true {
		vals = append(vals, words[0]+"%")
	} else {
		vals = append(vals, words[0])
	}

	for i, word := range words {
		if i == 0 {
			continue
		}
		likes += "OR word LIKE ? "
		if all == true {
			vals = append(vals, word+"%")
		} else {
			vals = append(vals, word)
		}
	}

	rows, err := dictConn.Query("SELECT word, confidence FROM words WHERE word LIKE ? "+likes+" ORDER BY confidence DESC LIMIT 10", vals...)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	var results []Suggestion

	for rows.Next() {
		var item Suggestion
		rows.Scan(&item.word, &item.weight)
		results = append(results, item)
	}

	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}

	return results
}

func getFromDictionary(tokens []Token) DictionaryResult {
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
					till := result.word
					tillWeight := result.weight

					if tillWeight == -1 {
						continue
					}

					firstToken := t.token[0]
					results[j].word += firstToken.value1
					results[j].weight -= firstToken.weight

					search := []string{results[j].word}
					searchResults := searchDictionary(search, false)

					if len(searchResults) > 0 {
						tempFoundDictWords = append(tempFoundDictWords, searchResults[0])
					} else {
						// No need of processing this anymore
						results[j].weight = -1
					}

					for k, possibility := range t.token {
						if k == 0 {
							continue
						}

						newTill := till + possibility.value1

						search = []string{newTill}
						searchResults = searchDictionary(search, false)

						if len(searchResults) > 0 {
							tempFoundDictWords = append(tempFoundDictWords, searchResults[0])

							newWeight := tillWeight - possibility.weight

							sug := Suggestion{newTill, newWeight}
							results = append(results, sug)
						} else {
							result.weight = -1
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

func getMoreFromDictionary(words []Suggestion) [][]Suggestion {
	var results [][]Suggestion
	for _, sug := range words {
		search := []string{sug.word}
		searchResults := searchDictionary(search, true)
		results = append(results, searchResults)
	}
	return results
}

func getFromPatternDictionary(pattern string) []Suggestion {
	rows, err := dictConn.Query("SELECT word, confidence FROM words WHERE id IN (SELECT word_id FROM patterns_content WHERE pattern LIKE ?) ORDER BY confidence DESC LIMIT 10", pattern+"%")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	var results []Suggestion

	for rows.Next() {
		var item Suggestion
		rows.Scan(&item.word, &item.weight)
		item.weight += VARNAM_LEARNT_WORD_MIN_CONFIDENCE
		results = append(results, item)
	}

	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}

	return results
}
