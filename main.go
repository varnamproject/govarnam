package main

import (
	sql "database/sql"
	"fmt"
	"log"
	"os"
	"sort"

	_ "github.com/mattn/go-sqlite3"
)

var (
	vstConn  *sql.DB
	dictConn *sql.DB
)

// Token info for making a suggestion
type Token struct {
	tokenType int
	token     []Symbol
	position  int
}

// Symbol result from VST
type Symbol struct {
	id              int
	generalType     int
	matchType       int
	pattern         string
	value1          string
	value2          string
	value3          string
	tag             string
	weight          int
	priority        int
	acceptCondition int
	flags           int
}

// Suggestion suggestion
type Suggestion struct {
	word   string
	weight int
}

// DictionaryResult result from dictionary search
type DictionaryResult struct {
	sugs                 []Suggestion
	exactMatch           bool
	longestMatchPosition int
}

func openVST() {
	var err error
	vstConn, err = sql.Open("sqlite3", "./ml.vst")
	if err != nil {
		log.Fatal(err)
	}
}

func openDict() {
	var err error
	dictConn, err = sql.Open("sqlite3", "./ml.vst.learnings")
	if err != nil {
		log.Fatal(err)
	}
}

func search(ch string) []Symbol {
	rows, err := vstConn.Query("select id, type, match_type, pattern, value1, value2, value3, tag, weight, priority, accept_condition, flags from symbols where pattern = ?", ch)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	var results []Symbol

	for rows.Next() {
		var item Symbol
		rows.Scan(&item.id, &item.generalType, &item.matchType, &item.pattern, &item.value1, &item.value2, &item.value3, &item.tag, &item.weight, &item.priority, &item.acceptCondition, &item.flags)
		results = append(results, item)
	}

	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}

	return results
}

func tokenizeWord(word string) []Token {
	var results []Token

	var prevSequenceMatches []Symbol
	var sequence string

	i := 0
	for i < len(word) {
		ch := string(word[i])

		sequence += ch

		matches := search(sequence)
		// fmt.Println(sequence, matches)

		if len(matches) == 0 {
			// No more matches

			if len(sequence) == 1 {
				// No matches for a single char, add it
				token := Token{VARNAM_TOKEN_CHAR, matches, i}
				results = append(results, token)
			} else {
				// Backtrack and add the previous sequence matches
				token := Token{VARNAM_TOKEN_SYMBOL, prevSequenceMatches, i - 1}
				results = append(results, token)
				i--
			}

			sequence = ""
		} else {
			if i == len(word)-1 {
				// Last character
				token := Token{VARNAM_TOKEN_SYMBOL, matches, i}
				results = append(results, token)
			} else {
				prevSequenceMatches = matches
			}
		}
		i++
	}
	return results
}

func flatten(tokens []Token) []Suggestion {
	var results []Suggestion

	for i, t := range tokens {
		if t.tokenType == VARNAM_TOKEN_SYMBOL {
			if i == 0 {
				for _, possibility := range t.token {
					sug := Suggestion{possibility.value1, possibility.weight}
					results = append(results, sug)
				}
			} else {
				for j, result := range results {
					till := result.word
					tillWeight := result.weight

					firstToken := t.token[0]
					results[j].word += firstToken.value1
					results[j].weight += firstToken.weight

					for k, possibility := range t.token {
						if k == 0 {
							continue
						}

						newTill := till + possibility.value1
						newWeight := tillWeight + possibility.weight

						sug := Suggestion{newTill, newWeight}
						results = append(results, sug)
					}
				}
			}
		}
	}

	return results
}

func getTokenizedSuggestions(tokens []Token) []Suggestion {
	sugs := flatten(tokens)
	sort.SliceStable(sugs, func(i, j int) bool {
		return sugs[i].weight < sugs[j].weight
	})
	return sugs
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

	rows, err := dictConn.Query("SELECT word, confidence FROM words WHERE word LIKE ? "+likes+" ORDER BY confidence DESC", vals...)
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
					sug := Suggestion{possibility.value1, possibility.weight}
					results = append(results, sug)
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
					results[j].weight += firstToken.weight

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

							newWeight := tillWeight + possibility.weight

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

func transliterate(word string) {
	tokens := tokenizeWord(word)
	sugs := getTokenizedSuggestions(tokens)
	fmt.Println(sugs)
	dictSugs := getFromDictionary(tokens)
	fmt.Println(dictSugs)
}

func main() {
	openVST()
	openDict()
	transliterate(os.Args[1])
	defer vstConn.Close()
}
