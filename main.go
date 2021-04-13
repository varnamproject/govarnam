package main

import (
	sql "database/sql"
	"flag"
	"fmt"
	"log"
	"sort"

	_ "github.com/mattn/go-sqlite3"
)

var (
	vstConn  *sql.DB
	dictConn *sql.DB
	debug    bool = false
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

func searchSymbol(ch string) []Symbol {
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

		matches := searchSymbol(sequence)

		if debug {
			fmt.Println(sequence, matches)
		}

		if len(matches) == 0 {
			// No more matches

			if len(sequence) == 1 {
				// No matches for a single char, add it
				token := Token{VARNAM_TOKEN_CHAR, matches, i}
				results = append(results, token)
			} else {
				// Backtrack and add the previous sequence matches
				i--
				token := Token{VARNAM_TOKEN_SYMBOL, prevSequenceMatches, i}
				results = append(results, token)
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

func calcNewWeight(weight int, symbolWeight int, tokensLength int, position int) int {
	return weight - symbolWeight + (tokensLength-position)*2
}

func tokensToSuggestions(tokens []Token, greedy int) []Suggestion {
	var results []Suggestion

	if greedy == 0 {
		greedy = 100
	}

	for i, t := range tokens {
		if t.tokenType == VARNAM_TOKEN_SYMBOL {
			if i == 0 {
				for _, possibility := range t.token {
					sug := Suggestion{possibility.value1, VARNAM_TOKEN_BASIC_WEIGHT - possibility.weight}
					results = append(results, sug)
				}
			} else {
				for j, result := range results {
					till := result.word
					tillWeight := result.weight

					firstToken := t.token[0]
					results[j].word += firstToken.value1
					results[j].weight = calcNewWeight(results[j].weight, firstToken.weight, len(tokens), i)

					for k, possibility := range t.token {
						if k == 0 {
							continue
						}

						newTill := till + possibility.value1
						newWeight := calcNewWeight(tillWeight, possibility.weight, len(tokens), i)

						sug := Suggestion{newTill, newWeight}
						results = append(results, sug)

						if len(results) > greedy {
							break
						}
					}
				}
			}
		}
	}

	return results
}

func sortSuggestions(sugs []Suggestion) []Suggestion {
	sort.SliceStable(sugs, func(i, j int) bool {
		return sugs[i].weight > sugs[j].weight
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

func transliterate(word string) []Suggestion {
	var results []Suggestion
	tokens := tokenizeWord(word)

	dictSugs := getFromDictionary(tokens)

	if debug {
		fmt.Println("Dictionary results:", dictSugs)
	}

	if len(dictSugs.sugs) > 0 {
		results = dictSugs.sugs

		if dictSugs.exactMatch == false {
			restOfWord := word[dictSugs.longestMatchPosition+1:]

			if debug {
				fmt.Printf("Tokenizing %s\n", restOfWord)
			}

			restOfWordTokens := tokenizeWord(restOfWord)
			restOfWordSugs := tokensToSuggestions(restOfWordTokens, 0)

			if debug {
				fmt.Println("Tokenized:", restOfWordSugs)
			}

			for j, result := range results {
				till := result.word
				tillWeight := result.weight

				firstSug := restOfWordSugs[0]
				results[j].word += firstSug.word
				results[j].weight += firstSug.weight

				for k, sug := range restOfWordSugs {
					if k == 0 {
						continue
					}
					sug := Suggestion{till + sug.word, tillWeight + sug.weight}
					results = append(results, sug)
				}
			}

			// Add greedy basic tokenized suggestions at the end
			sugs := tokensToSuggestions(tokens, 8)
			for _, sug := range sugs {
				results = append(results, sug)
			}
		} else {
			moreFromDict := getMoreFromDictionary(dictSugs.sugs)
			for _, sugSet := range moreFromDict {
				for _, sug := range sugSet {
					results = append(results, sug)
				}
			}
		}
	} else {
		sugs := tokensToSuggestions(tokens, 0)
		results = sugs
	}

	results = sortSuggestions(results)

	return results
}

func main() {
	openVST()
	openDict()

	debugTemp := flag.Bool("debug", false, "Debug")
	flag.Parse()
	debug = *debugTemp
	args := flag.Args()

	fmt.Println(transliterate(args[0]))

	defer vstConn.Close()
}
