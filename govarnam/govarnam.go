package govarnam

/**
 * govarnam - An Indian language transliteration library
 * Copyright Subin Siby, 2021
 * Licensed under AGPL-3.0-only
 */

import (
	"context"
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

// SchemeInfo of VST
type SchemeInfo struct {
	SchemeID     string
	LangCode     string
	DisplayName  string
	Author       string
	CompiledDate string
}

// Varnam config
type Varnam struct {
	vstConn    *sql.DB
	dictConn   *sql.DB
	LangRules  LangRules
	SchemeInfo SchemeInfo
	Debug      bool

	// Maximum suggestions to obtain from dictionary
	DictionarySuggestionsLimit int

	// Maximum suggestions to be made from tokenizer
	TokenizerSuggestionsLimit int

	// Always include tokenizer made suggestions.
	// This may give bad results and suggestion list will be long
	TokenizerSuggestionsAlways bool

	// See setDefaultConfig() for the default values
}

// Suggestion suggestion
type Suggestion struct {
	Word      string
	Weight    int
	LearnedOn int
}

// TransliterationResult result
type TransliterationResult struct {
	// Exact matches found in dictionary if any
	// From both patterns and normal dict
	ExactMatches []Suggestion

	// Possible words matching from dictionary
	DictionarySuggestions []Suggestion

	// Possible words matching from patterns dictionary
	PatternDictionarySuggestions []Suggestion

	// All possible matches from tokenizer (VARNAM_MATCH_ALL)
	// Has a limit. The first few results will be VARNAM_MATCH_EXACT
	TokenizerSuggestions []Suggestion

	// VARNAM_MATCH_EXACT results from tokenizer.
	// No limit, mostly gives 1 or less than 3 outputs
	GreedyTokenized []Suggestion
}

/**
 * Convert tokens into suggestions.
 * partial - set true if only a part of a word is being tokenized and not an entire word
 */
func (varnam *Varnam) tokensToSuggestions(ctx context.Context, tokens []Token, partial bool) []Suggestion {
	var results []Suggestion

	select {
	case <-ctx.Done():
		return results

	default:
		// First, remove less weighted symbols
		for i, token := range tokens {
			var reducedSymbols []Symbol
			for _, symbol := range token.symbols {
				// TODO should 0 be fixed for all languages ?
				// Because this may differ according to data source
				// from where symbol frequency was found out
				if getSymbolWeight(symbol) == 0 {
					break
				}
				reducedSymbols = append(reducedSymbols, symbol)
			}
			tokens[i].symbols = reducedSymbols
		}

		addWord := func(word []string, weight int) {
			// TODO avoid division, performance improvement ?
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

		i := k
		for i >= 0 {
			if tokens[i].tokenType == VARNAM_TOKEN_SYMBOL && len(tokens[i].symbols)-1 > tokenPositions[i] {
				k = i
				break
			}
			i--
		}

		for len(results) < varnam.TokenizerSuggestionsLimit {
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

					var (
						symbolValue  string
						symbolWeight int
					)

					if i == 0 {
						if partial {
							// Since partial, the first character is not
							// the first character of word
							symbolValue = getSymbolValue(symbol, 1)
							symbolWeight = getSymbolWeight(symbol)
						} else {
							symbolValue = getSymbolValue(symbol, 0)
							symbolWeight = getSymbolWeight(symbol)
						}
					} else if symbol.generalType == VARNAM_SYMBOL_VIRAMA {
						/*
							we are resolving a virama. If the output ends with a virama already,
							add a ZWNJ to it, so that following character will not be combined.
							If output does not end with virama, add a virama and ZWNJ
						*/
						previousCharacter := word[i-1]
						if previousCharacter == varnam.LangRules.Virama {
							symbolValue = ZWNJ
						} else {
							symbolValue = getSymbolValue(symbol, i) + ZWNJ
						}
					} else {
						symbolValue = getSymbolValue(symbol, i)
					}

					word[i] = symbolValue
					weight += symbolWeight
				} else if t.tokenType == VARNAM_TOKEN_CHAR {
					word[i] = t.character
				}
				i--
			}

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
}

func (varnam *Varnam) setDefaultConfig() {
	ctx := context.Background()

	varnam.DictionarySuggestionsLimit = 5
	varnam.TokenizerSuggestionsLimit = 10
	varnam.TokenizerSuggestionsAlways = true

	varnam.LangRules.IndicDigits = false
	varnam.LangRules.Virama = varnam.searchSymbol(ctx, "~", VARNAM_MATCH_EXACT)[0].value1
}

// SortSuggestions by confidence and learned on time
func SortSuggestions(sugs []Suggestion) []Suggestion {
	// TODO write tests
	sort.SliceStable(sugs, func(i, j int) bool {
		if (sugs[i].LearnedOn == 0 || sugs[j].LearnedOn == 0) && !(sugs[i].LearnedOn == 0 && sugs[j].LearnedOn == 0) {
			return sugs[i].LearnedOn > sugs[j].LearnedOn
		}
		return sugs[i].Weight > sugs[j].Weight
	})
	return sugs
}

// Returns tokens, exactMatches, dictionary results, greedy tokenized
func (varnam *Varnam) transliterate(ctx context.Context, word string) ([]Token, []Suggestion, []Suggestion, []Suggestion) {
	var (
		exactMatches     []Suggestion
		dictSugs         []Suggestion
		patternsDictSugs []Suggestion
		tokenizerSugs    []Suggestion
		greedyTokenized  []Suggestion
		tokens           []Token
	)

	tokensChan := make(chan []Token)
	go varnam.channelTokenizeWord(ctx, word, VARNAM_MATCH_ALL, tokensChan)

	select {
	case <-ctx.Done():
		return tokens, exactMatches, dictResults, greedyTokenized

	case tokens = <-tokensChan:
		if len(tokens) == 0 {
			return tokens, exactMatches, dictResults, greedyTokenized
		}

		/* Channels make things faster, getting from DB is time-consuming */

		dictSugsChan := make(chan channelDictionaryResult)
		patternDictSugsChan := make(chan channelDictionaryResult)
		greedyTokenizedChan := make(chan []Suggestion)

		go varnam.channelGetFromDictionary(ctx, word, tokens, dictSugsChan)
		go varnam.channelGetFromPatternDictionary(ctx, word, patternDictSugsChan)
		go varnam.channelTokensToGreedySuggestions(ctx, tokens, greedyTokenizedChan)

		select {
		case <-ctx.Done():
			return tokens, exactMatches, dictResults, greedyTokenized

		case channelDictResult := <-dictSugsChan:
			exactMatches = append(exactMatches, channelDictResult.exactMatches...)
			dictResults = append(dictResults, channelDictResult.suggestions...)

			channelPatternDictResult := <-patternDictSugsChan
			exactMatches = append(exactMatches, channelPatternDictResult.exactMatches...)
			dictResults = append(dictResults, channelPatternDictResult.suggestions...)

			// Add greedy tokenized suggestions. This will only give exact match (VARNAM_MATCH_EXACT) results
			greedyTokenizedResult := <-greedyTokenizedChan
			greedyTokenized = SortSuggestions(greedyTokenizedResult)

			return tokens, exactMatches, dictResults, greedyTokenized
		}
	}
}

// Transliterate a word with all possibilities as results
func (varnam *Varnam) Transliterate(word string) TransliterationResult {
	var result TransliterationResult

	ctx := context.Background()
	tokens, exactMatches, dictResults, greedyTokenized := varnam.transliterate(ctx, word)

	sugs := dictResults

	if len(tokens) != 0 {
		sugs = append(sugs, exactMatches...)

		if len(exactMatches) == 0 || varnam.TokenizerSuggestionsAlways {
			tokenSugs := varnam.tokensToSuggestions(ctx, tokens, false)
			sugs = append(sugs, tokenSugs...)
		}
	}

	result.ExactMatch = SortSuggestions(exactMatches)
	result.Suggestions = SortSuggestions(sugs)
	result.GreedyTokenized = SortSuggestions(greedyTokenized)

	return result
}

// TransliterateWithContext Use Go context
func (varnam *Varnam) TransliterateWithContext(ctx context.Context, word string, resultChannel chan<- TransliterationResult) {
	var result TransliterationResult

	select {
	case <-ctx.Done():
		return

	default:
		tokens, exactMatches, dictResults, greedyTokenized := varnam.transliterate(ctx, word)

		sugs := dictResults
		result.DictionaryResultCount = len(dictResults)

		if len(tokens) != 0 {
			sugs = append(sugs, exactMatches...)

			if len(exactMatches) == 0 || varnam.TokenizerSuggestionsAlways {
				tokenSugs := varnam.tokensToSuggestions(ctx, tokens, false)
				sugs = append(sugs, tokenSugs...)
			}
		}

		result.ExactMatch = SortSuggestions(exactMatches)
		result.Suggestions = SortSuggestions(sugs)
		result.GreedyTokenized = SortSuggestions(greedyTokenized)

		resultChannel <- result
	}
}

// TransliterateGreedy transliterate word without all possible suggestions in result
func (varnam *Varnam) TransliterateGreedy(word string) TransliterationResult {
	var result TransliterationResult

	ctx := context.Background()
	_, exactMatches, dictResults, greedyTokenized := varnam.transliterate(ctx, word)

	result.ExactMatch = SortSuggestions(exactMatches)
	result.Suggestions = SortSuggestions(dictResults)
	result.GreedyTokenized = SortSuggestions(greedyTokenized)
	result.DictionaryResultCount = len(dictResults)

	return result
}

// Init Initialize varnam
func Init(vstPath string, dictPath string) Varnam {
	varnam := Varnam{}
	varnam.openVST(vstPath)
	varnam.openDict(dictPath)
	varnam.setDefaultConfig()
	varnam.setSchemeInfo()
	return varnam
}

// InitFromID Init from ID. Scheme ID doesn't necessarily be a language code
func InitFromID(schemeID string) (*Varnam, error) {
	var (
		vstPath  string
		dictPath string
	)

	vstPath, err := findVSTPath(schemeID)
	if err != nil {
		return nil, err
	}

	dictPath = findLearningsFilePath(schemeID)
	if !fileExists(dictPath) {
		fmt.Printf("Making Varnam Learnings File at %s\n", dictPath)
		os.MkdirAll(path.Dir(dictPath), 0750)
		err := makeDictionary(dictPath)
		if err != nil {
			return nil, err
		}
	}

	varnam := Init(vstPath, dictPath)

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
