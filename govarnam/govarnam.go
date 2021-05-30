package govarnam

import (
	sql "database/sql"
	"fmt"
	"log"
	"sort"
)

// Varnam config
type Varnam struct {
	vstConn  *sql.DB
	dictConn *sql.DB
	debug    bool
}

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

func (varnam *Varnam) openVST(vstPath string) {
	var err error
	varnam.vstConn, err = sql.Open("sqlite3", vstPath)
	if err != nil {
		log.Fatal(err)
	}
}

// Checks if a symbol exist in VST
func (varnam *Varnam) symbolExist(ch string) bool {
	rows, err := varnam.vstConn.Query("SELECT COUNT(*) FROM symbols WHERE value1 = ?", ch)
	checkError(err)

	count := 0
	for rows.Next() {
		err := rows.Scan(&count)
		checkError(err)
	}
	return count != 0
}

// Split a word into conjuncts
func (varnam *Varnam) splitWordByConjunct(input string) []string {
	var results []string

	var prevSequenceMatch string
	var sequence string

	word := []rune(input)

	i := 0
	for i < len(word) {
		ch := string(word[i])

		sequence += ch

		if !varnam.symbolExist(sequence) {
			// No more matches

			if len(sequence) == 1 {
				// No matches for a single char, add it
				results = append(results, sequence)
			} else {
				// Backtrack and add the previous sequence matches
				i--
				results = append(results, prevSequenceMatch)
			}

			sequence = ""
		} else {
			if i == len(word)-1 {
				// Last character
				results = append(results, sequence)
			} else {
				prevSequenceMatch = sequence
			}
		}
		i++
	}
	return results
}

func (varnam *Varnam) searchSymbol(ch string, possibilityLimit int) []Symbol {
	rows, err := varnam.vstConn.Query("SELECT id, type, match_type, pattern, value1, value2, value3, tag, weight, priority, accept_condition, flags from symbols WHERE pattern = ? ORDER BY weight ASC LIMIT ?", ch, possibilityLimit)
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

func (varnam *Varnam) tokenizeWord(word string, possibilityLimit int) []Token {
	var results []Token

	var prevSequenceMatches []Symbol
	var sequence string

	i := 0
	for i < len(word) {
		ch := string(word[i])

		sequence += ch

		matches := varnam.searchSymbol(sequence, possibilityLimit)

		if varnam.debug {
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

func getNewValue(weight int, symbol Symbol, tokensLength int, position int) (string, int) {
	/**
	 * Weight priority:
	 * 1. Position of character in string
	 * 2. Symbol's probability occurence
	 */
	newWeight := weight - symbol.weight + (tokensLength-position)*2 + (VARNAM_MATCH_POSSIBILITY - symbol.matchType)

	var value string
	if symbol.generalType == VARNAM_SYMBOL_VOWEL && position > 0 {
		// If in between word, we use the vowel and not the consonant
		value = symbol.value2 // ാ
	} else {
		value = symbol.value1 // ആ
	}

	return value, newWeight
}

/**
 * greed - Set to true for getting only exact match suggestions.
 * partial - set true if only a part of a word is being tokenized and not an entire word
 */
func tokensToSuggestions(tokens []Token, greedy bool, partial bool) []Suggestion {
	var results []Suggestion

	for i, t := range tokens {
		if t.tokenType == VARNAM_TOKEN_SYMBOL {
			if i == 0 {
				for _, possibility := range t.token {
					if greedy && possibility.matchType == VARNAM_MATCH_POSSIBILITY {
						continue
					}

					var value string
					if partial && possibility.generalType == VARNAM_SYMBOL_VOWEL {
						value = possibility.value2
					} else {
						value = possibility.value1
					}

					sug := Suggestion{value, VARNAM_TOKEN_BASIC_WEIGHT - possibility.weight}
					results = append(results, sug)
				}
			} else {
				for j, result := range results {
					till := result.word
					tillWeight := result.weight

					firstToken := t.token[0]

					newValue, newWeight := getNewValue(results[j].weight, firstToken, len(tokens), i)

					results[j].word += newValue
					results[j].weight = newWeight

					for k, possibility := range t.token {
						if k == 0 || (greedy && possibility.matchType == VARNAM_MATCH_POSSIBILITY) {
							continue
						}

						newValue, newWeight := getNewValue(tillWeight, possibility, len(tokens), i)

						newTill := till + newValue

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

// Transliterate a word
func (varnam *Varnam) Transliterate(word string, possibilityLimit int) TransliterationResult {
	var (
		results               []Suggestion
		transliterationResult TransliterationResult
	)

	tokens := varnam.tokenizeWord(word, 10)

	dictSugs := varnam.getFromDictionary(tokens)

	if varnam.debug {
		fmt.Println("Dictionary results:", dictSugs)
	}

	if len(dictSugs.sugs) > 0 {
		results = append(results, dictSugs.sugs...)

		// Add greedy tokenized suggestions. This will give >=1 and <5 suggestions
		transliterationResult.greedyTokenized = tokensToSuggestions(tokens, true, false)

		if dictSugs.exactMatch == false {
			restOfWord := word[dictSugs.longestMatchPosition+1:]

			if varnam.debug {
				fmt.Printf("Tokenizing %s\n", restOfWord)
			}

			restOfWordTokens := varnam.tokenizeWord(restOfWord, possibilityLimit)
			restOfWordSugs := tokensToSuggestions(restOfWordTokens, false, true)

			if varnam.debug {
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
			moreFromDict := varnam.getMoreFromDictionary(dictSugs.sugs)
			for _, sugSet := range moreFromDict {
				for _, sug := range sugSet {
					results = append(results, sug)
				}
			}
		}
	} else {
		sugs := tokensToSuggestions(tokens, false, false)
		results = sugs
	}

	patternDictSugs := varnam.getFromPatternDictionary(word)
	if len(patternDictSugs) > 0 {
		results = append(results, patternDictSugs...)

		if varnam.debug {
			fmt.Println("Pattern dictionary results:", patternDictSugs)
		}
	}

	results = sortSuggestions(results)
	transliterationResult.suggestions = results

	return transliterationResult
}

// Init Initialize varnam
func Init(vstPath string, dictPath string) Varnam {
	varnam := Varnam{}
	varnam.openVST(vstPath)
	varnam.openDict(dictPath)
	return varnam
}

// Debug turn on or off debug messages
func (varnam *Varnam) Debug(val bool) {
	varnam.debug = val
}

// Close close db connections
func (varnam *Varnam) Close() {
	defer varnam.vstConn.Close()
	defer varnam.dictConn.Close()
}
