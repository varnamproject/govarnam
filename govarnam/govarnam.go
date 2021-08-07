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
	"log"
	"os"
	"sort"
	"strings"
	"time"

	// sqlite3
	_ "github.com/mattn/go-sqlite3"
)

// LangRules language reulated config
type LangRules struct {
	Virama               string
	IndicDigits          bool
	PatternLongestLength int // Longest length of pattern in VST
}

// SchemeDetails of VST
type SchemeDetails struct {
	Identifier   string
	LangCode     string
	DisplayName  string
	Author       string
	CompiledDate string
}

// Varnam config
type Varnam struct {
	vstConn       *sql.DB
	dictConn      *sql.DB
	LangRules     LangRules
	SchemeDetails SchemeDetails
	Debug         bool

	PatternWordPartializers []func(*Suggestion)

	// Maximum suggestions to obtain from dictionary
	DictionarySuggestionsLimit int

	// Maximum suggestions to obtain from patterns dictionary
	PatternDictionarySuggestionsLimit int

	// Maximum suggestions to be made from tokenizer
	TokenizerSuggestionsLimit int

	// Always include tokenizer made suggestions.
	// Tokenizer results are not exactly the best, but it's alright
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
	// Has a limit. The first few results will be VARNAM_MATCH_EXACT.
	// This will only be filled if there are no exact matches.
	// Related: See Config.TokenizerSuggestionsAlways
	TokenizerSuggestions []Suggestion

	// VARNAM_MATCH_EXACT results from tokenizer.
	// No limit, mostly gives 1 or less than 3 outputs
	GreedyTokenized []Suggestion
}

/**
 * Convert tokens into suggestions.
 * partial - set true if only a part of a word is being tokenized and not an entire word
 */
func (varnam *Varnam) tokensToSuggestions(ctx context.Context, tokensPointer *[]Token, partial bool, limit int) []Suggestion {
	var results []Suggestion
	tokens := *tokensPointer

	select {
	case <-ctx.Done():
		return results

	default:
		tokens = removeLessWeightedSymbols(tokens)

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

		for len(results) < limit {
			// One loop will make one word
			word := make([]string, len(tokens))
			weight := 0

			// i is the character position we're making
			i := len(tokens) - 1
			for i >= 0 {
				t := tokens[i]
				if t.tokenType == VARNAM_TOKEN_SYMBOL {
					symbol := t.symbols[tokenPositions[i]]

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
						symbolWeight = getSymbolWeight(symbol)
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
	varnam.PatternDictionarySuggestionsLimit = 10

	varnam.TokenizerSuggestionsLimit = 10
	varnam.TokenizerSuggestionsAlways = true

	varnam.LangRules.IndicDigits = false
	varnam.LangRules.Virama = varnam.searchSymbol(ctx, "~", VARNAM_MATCH_EXACT, VARNAM_TOKEN_ACCEPT_ALL)[0].value1

	if varnam.SchemeDetails.LangCode == "ml" {
		varnam.RegisterPatternWordPartializer(varnam.mlPatternWordPartializer)
	}
}

// SortSuggestions by weight and learned on time
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

// Returns tokens and all found suggestions
func (varnam *Varnam) transliterate(ctx context.Context, word string) (
	*[]Token,
	TransliterationResult) {
	var (
		result TransliterationResult
	)

	start := time.Now()

	tokensPointerChan := make(chan *[]Token)
	go varnam.channelTokenizeWord(ctx, word, VARNAM_MATCH_ALL, false, tokensPointerChan)

	select {
	case <-ctx.Done():
		return nil, result

	case tokensPointer := <-tokensPointerChan:
		if len(*tokensPointer) == 0 {
			return nil, result
		}

		if varnam.Debug {
			fmt.Println(*tokensPointer)
		}

		/* Channels make things faster, getting from DB is time-consuming */

		dictSugsChan := make(chan channelDictionaryResult)
		patternDictSugsChan := make(chan channelDictionaryResult)
		greedyTokenizedChan := make(chan []Suggestion)

		go varnam.channelGetFromDictionary(ctx, word, tokensPointer, dictSugsChan)
		go varnam.channelGetFromPatternDictionary(ctx, word, patternDictSugsChan)
		go varnam.channelTokensToGreedySuggestions(ctx, tokensPointer, greedyTokenizedChan)

		tokenizerSugsChan := make(chan []Suggestion)
		tokenizerSugsCalled := false

		select {
		case <-ctx.Done():
			return nil, result

		case channelDictResult := <-dictSugsChan:
			// From dictionary
			result.ExactMatches = channelDictResult.exactMatches
			result.DictionarySuggestions = channelDictResult.suggestions

			select {
			case <-ctx.Done():
				return nil, result
			case channelPatternDictResult := <-patternDictSugsChan:
				// From patterns dictionary
				result.ExactMatches = append(result.ExactMatches, channelPatternDictResult.exactMatches...)
				result.PatternDictionarySuggestions = SortSuggestions(channelPatternDictResult.suggestions)

				if len(result.ExactMatches) == 0 || varnam.TokenizerSuggestionsAlways {
					go varnam.channelTokensToSuggestions(ctx, tokensPointer, varnam.TokenizerSuggestionsLimit, tokenizerSugsChan)
					tokenizerSugsCalled = true
				}

				select {
				case <-ctx.Done():
					return nil, result

				// Add greedy tokenized suggestions. This will only give exact match (VARNAM_MATCH_EXACT) results
				case greedyTokenizedResult := <-greedyTokenizedChan:
					result.GreedyTokenized = SortSuggestions(greedyTokenizedResult)

					// Sort everything now

					result.ExactMatches = SortSuggestions(result.ExactMatches)
					result.DictionarySuggestions = SortSuggestions(result.DictionarySuggestions)
					result.PatternDictionarySuggestions = SortSuggestions(result.PatternDictionarySuggestions)

					if tokenizerSugsCalled {
						select {
						case <-ctx.Done():
							return nil, result

						case tokenizerSugs := <-tokenizerSugsChan:
							result.TokenizerSuggestions = SortSuggestions(tokenizerSugs)

							if LOG_TIME_TAKEN {
								log.Printf("%s took %v\n", "transliteration", time.Since(start))
							}

							return tokensPointer, result
						}

					} else {
						if LOG_TIME_TAKEN {
							log.Printf("%s took %v\n", "transliteration", time.Since(start))
						}

						return tokensPointer, result
					}
				}
			}
		}
	}
}

// Transliterate a word with all possibilities as results
func (varnam *Varnam) Transliterate(word string) TransliterationResult {
	ctx := context.Background()
	_, result := varnam.transliterate(ctx, word)
	return result
}

// TransliterateWithContext Use Go context
func (varnam *Varnam) TransliterateWithContext(ctx context.Context, word string, resultChannel chan<- TransliterationResult) {
	select {
	case <-ctx.Done():
		return

	default:
		_, result := varnam.transliterate(ctx, word)
		resultChannel <- result
		close(resultChannel)
	}
}

// TransliterateGreedy transliterate word without all possible suggestions in result
func (varnam *Varnam) TransliterateGreedy(word string) TransliterationResult {
	ctx := context.Background()
	_, result := varnam.transliterate(ctx, word)

	return result
}

// ReverseTransliterate do a reverse transliteration
func (varnam *Varnam) ReverseTransliterate(word string) ([]Suggestion, error) {
	var results []Suggestion
	ctx := context.Background()

	tokens := varnam.splitTextByConjunct(ctx, word)

	for i, token := range tokens {
		for j, symbol := range token.symbols {
			tokens[i].symbols[j].value1 = symbol.pattern
			tokens[i].symbols[j].value2 = symbol.pattern
		}
	}

	results = SortSuggestions(varnam.tokensToSuggestions(ctx, &tokens, false, varnam.TokenizerSuggestionsLimit))

	return results, nil
}

// RegisterPatternWordPartializer A word partializer remove word ending
// with proper alternative so that the word can be tokenized further.
// Useful for malayalam to replace last chil letter with its root
func (varnam *Varnam) RegisterPatternWordPartializer(cb func(*Suggestion)) {
	varnam.PatternWordPartializers = append(varnam.PatternWordPartializers, cb)
}

// Init Initialize varnam. Dictionary will be created if it doesn't exist
func Init(vstPath string, dictPath string) (*Varnam, error) {
	varnam := Varnam{}

	err := varnam.InitVST(vstPath)
	if err != nil {
		return nil, err
	}
	err = varnam.InitDict(dictPath)
	if err != nil {
		return nil, err
	}

	varnam.setDefaultConfig()

	return &varnam, nil
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

	varnam := Varnam{}
	varnam.InitVST(vstPath)

	// One dictionary for one language, not for different scheme
	dictPath = findLearningsFilePath(varnam.SchemeDetails.LangCode)

	err = varnam.InitDict(dictPath)
	if err != nil {
		return nil, err
	}

	varnam.setDefaultConfig()

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
func (varnam *Varnam) Close() error {
	varnam.vstConn.Close()
	varnam.dictConn.Close()
	return nil
}
