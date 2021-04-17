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
	vstConn *sql.DB
	debug   bool = false
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

type TransliterationResult struct {
	suggestions     []Suggestion
	greedyTokenized []Suggestion
}

func openVST() {
	var err error
	vstConn, err = sql.Open("sqlite3", "./ml.vst")
	if err != nil {
		log.Fatal(err)
	}
}

func searchSymbol(ch string, possibilityLimit int) []Symbol {
	rows, err := vstConn.Query("SELECT id, type, match_type, pattern, value1, value2, value3, tag, weight, priority, accept_condition, flags from symbols WHERE pattern = ? ORDER BY weight ASC LIMIT ?", ch, possibilityLimit)
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

func tokenizeWord(word string, possibilityLimit int) []Token {
	var results []Token

	var prevSequenceMatches []Symbol
	var sequence string

	i := 0
	for i < len(word) {
		ch := string(word[i])

		sequence += ch

		matches := searchSymbol(sequence, possibilityLimit)

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

func calcNewWeight(weight int, symbol Symbol, tokensLength int, position int) int {
	/**
	 * Weight priority:
	 * 1. Position of character in string
	 * 2. Symbol's probability occurence
	 */
	return weight - symbol.weight + (tokensLength-position)*2 + (VARNAM_MATCH_POSSIBILITY - symbol.matchType)
}

func tokensToSuggestions(tokens []Token, greedy bool) []Suggestion {
	var results []Suggestion

	for i, t := range tokens {
		if t.tokenType == VARNAM_TOKEN_SYMBOL {
			if i == 0 {
				for _, possibility := range t.token {
					if greedy && possibility.matchType == VARNAM_MATCH_POSSIBILITY {
						continue
					}
					sug := Suggestion{possibility.value1, VARNAM_TOKEN_BASIC_WEIGHT - possibility.weight}
					results = append(results, sug)
				}
			} else {
				for j, result := range results {
					till := result.word
					tillWeight := result.weight

					firstToken := t.token[0]
					results[j].word += firstToken.value1
					results[j].weight = calcNewWeight(results[j].weight, firstToken, len(tokens), i)

					for k, possibility := range t.token {
						if k == 0 || (greedy && possibility.matchType == VARNAM_MATCH_POSSIBILITY) {
							continue
						}

						newTill := till + possibility.value1
						newWeight := calcNewWeight(tillWeight, possibility, len(tokens), i)

						sug := Suggestion{newTill, newWeight}
						results = append(results, sug)
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

func transliterate(word string, possibilityLimit int) TransliterationResult {
	var (
		results               []Suggestion
		transliterationResult TransliterationResult
	)
	tokens := tokenizeWord(word, 10)

	dictSugs := getFromDictionary(tokens)

	if debug {
		fmt.Println("Dictionary results:", dictSugs)
	}

	if len(dictSugs.sugs) > 0 {
		results = dictSugs.sugs

		// Add greedy tokenized suggestions. These will be >=1 and <5
		transliterationResult.greedyTokenized = tokensToSuggestions(tokens, true)

		if dictSugs.exactMatch == false {
			restOfWord := word[dictSugs.longestMatchPosition+1:]

			if debug {
				fmt.Printf("Tokenizing %s\n", restOfWord)
			}

			restOfWordTokens := tokenizeWord(restOfWord, possibilityLimit)
			restOfWordSugs := tokensToSuggestions(restOfWordTokens, false)

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
		} else {
			moreFromDict := getMoreFromDictionary(dictSugs.sugs)
			for _, sugSet := range moreFromDict {
				for _, sug := range sugSet {
					results = append(results, sug)
				}
			}
		}
	} else {
		sugs := tokensToSuggestions(tokens, false)
		results = sugs
	}

	results = sortSuggestions(results)
	transliterationResult.suggestions = results

	return transliterationResult
}

func main() {
	openVST()
	openDict()

	debugTemp := flag.Bool("debug", false, "Debug")
	flag.Parse()
	debug = *debugTemp
	args := flag.Args()

	fmt.Println(transliterate(args[0], 2))

	defer vstConn.Close()
}
