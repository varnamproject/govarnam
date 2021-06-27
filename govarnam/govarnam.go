package govarnam

import (
	sql "database/sql"
	"fmt"
	"os"
	"path"
	"sort"
	"strings"

	// sqlite3
	_ "github.com/mattn/go-sqlite3"
)

// LangRules language reulated config
type LangRules struct {
	Virama      string
	IndicDigits bool
}

// Varnam config
type Varnam struct {
	vstConn          *sql.DB
	dictConn         *sql.DB
	LangRules        LangRules
	Debug            bool
	SuggestionsLimit int
}

// Suggestion suggestion
type Suggestion struct {
	Word      string
	Weight    int
	LearnedOn int
}

// TransliterationResult result
type TransliterationResult struct {
	ExactMatch      []Suggestion
	Suggestions     []Suggestion
	GreedyTokenized []Suggestion
}

func getWeight(symbol Symbol) int {
	return symbol.weight + (VARNAM_MATCH_POSSIBILITY-symbol.matchType)*100
}

func (varnam *Varnam) getNewValueAndWeight(weight int, symbol Symbol, previousCharacter string, position int) (string, int) {
	/**
	 * Weight priority:
	 * 1. Position of character in string
	 * 2. Symbol's probability occurence
	 */
	newWeight := weight + getWeight(symbol)

	var value string

	if symbol.generalType == VARNAM_SYMBOL_VIRAMA {
		/*
			we are resolving a virama. If the output ends with a virama already,
			add a ZWNJ to it, so that following character will not be combined.
			If output does not end with virama, add a virama and ZWNJ
		*/
		if previousCharacter == varnam.LangRules.Virama {
			value = ZWNJ
		} else {
			value = getSymbolValue(symbol, position) + ZWNJ
		}
	} else {
		value = getSymbolValue(symbol, position)
	}

	return value, newWeight
}

/**
 * greed - Set to true for getting only VARNAM_MATCH_EXACT suggestions.
 * partial - set true if only a part of a word is being tokenized and not an entire word
 */
func (varnam *Varnam) tokensToSuggestions(inputTokens []Token, partial bool) []Suggestion {
	var results []Suggestion

	tokens := make([]Token, len(inputTokens))
	copy(tokens, inputTokens)

	// First, remove less weighted symbols
	for i, token := range tokens {
		var reducedSymbols []Symbol
		for _, symbol := range token.symbols {
			reducedSymbols = append(reducedSymbols, symbol)

			// TODO should 10% be fixed for all languages ?
			// Because this may differ according to data source
			// from where symbol frequency was found out
			if getWeight(symbol) < 10 {
				break
			}
		}
		tokens[i].symbols = reducedSymbols
	}

	addWord := func(word []string, weight int) {
		weight = weight / 100
		results = append(results, Suggestion{strings.Join(word, ""), weight, 0})
	}

	// Tracks index of each token possibilities
	// -----
	// Suppose input is "vardhichu". We will try each possibilities of each token
	// The index of these possibilities is tracked here
	// [0 0 0 0] => വ ർ ധി ചു
	// [0 0 0 1] => വ ർ ധി ച്ചു
	// [0 0 1 0] => വ ർ ഥി ചു
	// [0 0 1 1] => വ ർ ഥി ച്ചു
	tokenPositions := make([]int, len(tokens))

	// We go right to left.
	// We try possibilities from the last character (k) where there are multiple possibilities.
	// if it's over we shift the possibility on left, so on and on
	k := len(tokens) - 1

	i := 0
	for i >= 0 {
		if tokens[i].tokenType == VARNAM_TOKEN_SYMBOL && len(tokens[i].symbols)-1 > tokenPositions[i] {
			k = i
			break
		}
		i--
	}

	for len(results) < varnam.SuggestionsLimit {
		// One loop will make one word
		word := make([]string, len(tokens))
		weight := 0

		// i is the character position we're making
		i := len(tokens) - 1
		for i >= 0 {
			t := tokens[i]
			if t.tokenType == VARNAM_TOKEN_SYMBOL {
				symbol := t.symbols[tokenPositions[i]]

				if symbol.acceptCondition != VARNAM_TOKEN_ACCEPT_ALL {
					var state int
					if i == 0 {
						state = VARNAM_TOKEN_ACCEPT_IF_STARTS_WITH
					} else if i == len(tokens)-1 {
						state = VARNAM_TOKEN_ACCEPT_IF_ENDS_WITH
					} else {
						state = VARNAM_TOKEN_ACCEPT_IF_IN_BETWEEN
					}
					if symbol.acceptCondition != state {
						i--
						continue
					}
				}

				symbolWeight := getWeight(symbol)

				var value string
				if partial {
					value = getSymbolValue(symbol, 1)
				} else {
					value = getSymbolValue(symbol, 0)
				}

				word[i] = value
				weight += symbolWeight
			} else if t.tokenType == VARNAM_TOKEN_CHAR {
				word[i] = *t.character
			}
			i--
		}

		fmt.Println(k, tokenPositions, tokens[k].symbols)

		// If no more possibilites, go to the next one
		if tokenPositions[k] >= len(tokens[k].symbols)-1 {
			// Reset the currently permuted position
			tokenPositions[k] = 0

			// Find the next place where there are more possibilities
			i := k - 1
			for i >= 0 {
				if tokens[i].tokenType == VARNAM_TOKEN_SYMBOL && len(tokens[i].symbols)-1 > tokenPositions[i] {
					// Set the newly gonna permuting position
					tokenPositions[i]++
					break
				} else {
					tokenPositions[i] = 0
				}
				i--
			}
			addWord(word, weight)
			if i < 0 {
				break
			}
		} else {
			tokenPositions[k]++
			addWord(word, weight)
		}
	}

	return results
}

func (varnam *Varnam) setDefaultConfig() {
	varnam.SuggestionsLimit = 10
	varnam.LangRules.IndicDigits = false
	varnam.LangRules.Virama = varnam.searchSymbol("~", VARNAM_MATCH_EXACT)[0].value1
}

func sortSuggestions(sugs []Suggestion) []Suggestion {
	sort.SliceStable(sugs, func(i, j int) bool {
		if (sugs[i].LearnedOn == 0 || sugs[j].LearnedOn == 0) && !(sugs[i].LearnedOn == 0 && sugs[j].LearnedOn == 0) {
			return sugs[i].LearnedOn > sugs[j].LearnedOn
		}
		return sugs[i].Weight > sugs[j].Weight
	})
	return sugs
}

// Returns tokens, exactMatches, dictionary results, greedy tokenized
func (varnam *Varnam) transliterate(word string) ([]Token, []Suggestion, []Suggestion, []Suggestion) {
	var (
		dictResults  []Suggestion
		exactMatches []Suggestion
	)

	tokens := varnam.tokenizeWord(word, VARNAM_MATCH_ALL)

	/* Channels make things faster, getting from DB is time-consuming */

	dictSugsChan := make(chan DictionaryResult)
	patternDictSugsChan := make(chan []PatternDictionarySuggestion)
	greedyTokenizedChan := make(chan []Suggestion)

	moreFromDictChan := make(chan [][]Suggestion)
	triggeredGetMoreFromDict := false

	go varnam.channelGetFromDictionary(tokens, dictSugsChan)
	go varnam.channelGetFromPatternDictionary(word, patternDictSugsChan)
	go varnam.channelTokensToGreedySuggestions(tokens, greedyTokenizedChan)

	dictSugs := <-dictSugsChan

	if varnam.Debug {
		fmt.Println("Dictionary results:", dictSugs)
	}

	if len(dictSugs.sugs) > 0 {
		if dictSugs.exactMatch == false {
			// These will be partial words
			restOfWord := word[dictSugs.longestMatchPosition+1:]
			dictResults = varnam.tokenizeRestOfWord(restOfWord, dictSugs.sugs)
		} else {
			exactMatches = dictSugs.sugs

			// Since partial words are in dictionary, exactMatch will be TRUE
			// for pathway to a word. Hence we're calling this here
			go varnam.channelGetMoreFromDictionary(dictSugs.sugs, moreFromDictChan)
			triggeredGetMoreFromDict = true
		}
	}

	patternDictSugs := <-patternDictSugsChan

	if len(patternDictSugs) > 0 {
		if varnam.Debug {
			fmt.Println("Pattern dictionary results:", patternDictSugs)
		}

		for _, match := range patternDictSugs {
			if match.Length < len(word) {
				restOfWord := word[match.Length:]
				filled := varnam.tokenizeRestOfWord(restOfWord, []Suggestion{match.Sug})
				dictResults = append(dictResults, filled...)
			} else if match.Length == len(word) {
				// Same length
				exactMatches = append(exactMatches, match.Sug)
			} else {
				dictResults = append(dictResults, match.Sug)
			}
		}
	}

	if triggeredGetMoreFromDict {
		moreFromDict := <-moreFromDictChan

		if varnam.Debug {
			fmt.Println("More dictionary results:", moreFromDict)
		}

		for _, sugSet := range moreFromDict {
			dictResults = append(dictResults, sugSet...)
		}
	}

	// Add greedy tokenized suggestions. This will only give exact match (VARNAM_MATCH_EXACT) results
	greedyTokenized := sortSuggestions(<-greedyTokenizedChan)

	return tokens, exactMatches, dictResults, greedyTokenized
}

// Transliterate a word with all possibilities as results
func (varnam *Varnam) Transliterate(word string) TransliterationResult {
	var result TransliterationResult

	tokens, exactMatches, dictResults, greedyTokenized := varnam.transliterate(word)

	sugs := dictResults

	if len(exactMatches) == 0 {
		tokenSugs := varnam.tokensToSuggestions(tokens, false)
		sugs = append(sugs, tokenSugs...)
	} else {
		sugs = append(sugs, exactMatches...)
	}

	result.ExactMatch = sortSuggestions(exactMatches)
	result.Suggestions = sortSuggestions(sugs)
	result.GreedyTokenized = sortSuggestions(greedyTokenized)

	return result
}

// TransliterateGreedy transliterate word without all possible suggestions in result
func (varnam *Varnam) TransliterateGreedy(word string) TransliterationResult {
	var result TransliterationResult

	_, exactMatches, dictResults, greedyTokenized := varnam.transliterate(word)

	result.ExactMatch = sortSuggestions(exactMatches)
	result.Suggestions = sortSuggestions(dictResults)
	result.GreedyTokenized = sortSuggestions(greedyTokenized)

	return result
}

// Init Initialize varnam
func Init(vstPath string, dictPath string) Varnam {
	varnam := Varnam{}
	varnam.openVST(vstPath)
	varnam.openDict(dictPath)
	varnam.setDefaultConfig()
	return varnam
}

// InitFromLang code
func InitFromLang(langCode string) (*Varnam, error) {
	var (
		vstPath  *string = nil
		dictPath string
	)

	vstPath = findVSTPath(langCode)

	dictPath = findLearningsFilePath(langCode)
	if !fileExists(dictPath) {
		fmt.Printf("Making Varnam Learnings File at %s\n", dictPath)
		os.MkdirAll(path.Dir(dictPath), 0750)
		makeDictionary(dictPath)
	}

	if vstPath == nil {
		return nil, fmt.Errorf("Couldn't find VST")
	}

	varnam := Init(*vstPath, dictPath)

	return &varnam, nil
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func dirExists(loc string) bool {
	info, err := os.Stat(loc)
	if os.IsNotExist(err) {
		return false
	}
	return info.IsDir()
}

// Close close db connections
func (varnam *Varnam) Close() {
	defer varnam.vstConn.Close()
	defer varnam.dictConn.Close()
}
