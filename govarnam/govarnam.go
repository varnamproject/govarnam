package govarnam

/**
 * govarnam - An Indian language transliteration library
 * Copyright Subin Siby <mail at subinsb (.) com>, 2021
 * Licensed under AGPL-3.0-only. See LICENSE.txt
 */

import (
	"context"
	sql "database/sql"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

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
	IsStable     bool
}

type VSTMakerConfig struct {
	// Not a config. State variable
	Buffering bool

	IgnoreDuplicateTokens bool
	UseDeadConsonants     bool
}

// Varnam config
type Varnam struct {
	VSTPath  string
	DictPath string

	vstConn  *sql.DB
	dictConn *sql.DB

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

	// Whether only exact scheme match should be considered
	// for dictionary search and discard possibility matches
	DictionaryMatchExact bool

	VSTMakerConfig VSTMakerConfig

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
	// Exactly found words in dictionary if there is any.
	// From both patterns and normal dict
	ExactWords []Suggestion

	// Exactly starting word matches in dictionary if there is any.
	// Not applicable for patterns dictionary.
	ExactMatches []Suggestion

	// Possible word suggestions from dictionary
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

func (varnam *Varnam) log(msg string) {
	if varnam.Debug {
		fmt.Println(msg)
	}
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
	varnam.DictionarySuggestionsLimit = 5
	varnam.PatternDictionarySuggestionsLimit = 5

	varnam.TokenizerSuggestionsLimit = 10
	varnam.TokenizerSuggestionsAlways = true

	varnam.DictionaryMatchExact = false

	varnam.LangRules.IndicDigits = false

	varnam.LangRules.Virama, _ = varnam.getVirama()

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

		// Only exact tokens
		exactTokens := make([]Token, len(*tokensPointer))
		copy(exactTokens, *tokensPointer)

		exactTokens = removeNonExactTokens(exactTokens)

		if varnam.DictionaryMatchExact {
			go varnam.channelGetFromDictionary(ctx, word, &exactTokens, dictSugsChan)
		} else {
			go varnam.channelGetFromDictionary(ctx, word, tokensPointer, dictSugsChan)
		}

		go varnam.channelGetFromPatternDictionary(ctx, word, patternDictSugsChan)
		go varnam.channelTokensToGreedySuggestions(ctx, &exactTokens, greedyTokenizedChan)

		tokenizerSugsChan := make(chan []Suggestion)
		tokenizerSugsCalled := false

		select {
		case <-ctx.Done():
			return nil, result

		case channelDictResult := <-dictSugsChan:
			// From dictionary
			result.ExactWords = channelDictResult.exactWords
			result.ExactMatches = channelDictResult.exactMatches
			result.DictionarySuggestions = channelDictResult.suggestions

			select {
			case <-ctx.Done():
				return nil, result
			case channelPatternDictResult := <-patternDictSugsChan:
				// From patterns dictionary
				result.ExactWords = append(result.ExactWords, channelPatternDictResult.exactWords...)
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

					result.ExactWords = SortSuggestions(result.ExactWords)
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

// TransliterateAdvanced transliterate with a detailed structure as result
func (varnam *Varnam) TransliterateAdvanced(word string) TransliterationResult {
	ctx := context.Background()
	_, result := varnam.transliterate(ctx, word)
	return result
}

// TransliterateAdvancedWithContext transliterate with a detailed structure as result Go context
func (varnam *Varnam) TransliterateAdvancedWithContext(ctx context.Context, word string, resultChannel chan<- TransliterationResult) {
	select {
	case <-ctx.Done():
		return

	default:
		_, result := varnam.transliterate(ctx, word)
		resultChannel <- result
		close(resultChannel)
	}
}

// Flatten TransliterationResult struct to a suggestion array
func flattenTR(result TransliterationResult) []Suggestion {
	var combined []Suggestion

	dictCombined := result.ExactWords

	if len(result.ExactWords) == 0 {
		dictCombined = append(dictCombined, result.ExactMatches...)
	}

	dictCombined = append(dictCombined, result.PatternDictionarySuggestions...)
	dictCombined = append(dictCombined, result.DictionarySuggestions...)

	/**
	 * Show greedy tokenized first if length less than 3
	 */
	if len(result.GreedyTokenized) > 0 && utf8.RuneCountInString(result.GreedyTokenized[0].Word) < 3 {
		combined = append(combined, result.GreedyTokenized...)
		combined = append(combined, result.ExactWords...)
		combined = append(combined, result.ExactMatches...)
		combined = append(combined, result.PatternDictionarySuggestions...)
		combined = append(combined, result.DictionarySuggestions...)
	} else {
		/**
		 * Show greedy tokenized always at 2nd
		 * And then rest of the results from exact matches or the 2 dictionary
		 * https://github.com/varnamproject/govarnam/issues/12
		 */

		if len(dictCombined) > 0 {
			combined = append(combined, dictCombined[0])
		}

		combined = append(combined, result.GreedyTokenized...)

		// Insert rest of them
		if len(dictCombined) > 1 {
			combined = append(combined, dictCombined[1:]...)
		}
	}

	combined = append(combined, result.TokenizerSuggestions...)
	combined = append(combined, result.GreedyTokenized...)
	return combined
}

// Transliterate transliterate with output array
func (varnam *Varnam) Transliterate(word string) []Suggestion {
	return flattenTR(varnam.TransliterateAdvanced(word))
}

// TransliterateWithContext Transliterate but with Go context
func (varnam *Varnam) TransliterateWithContext(ctx context.Context, word string, resultChannel chan<- []Suggestion) {
	select {
	case <-ctx.Done():
		return
	default:
		_, result := varnam.transliterate(ctx, word)
		resultChannel <- flattenTR(result)
		close(resultChannel)
	}
}

// TransliterateGreedyTokenized transliterate word, only tokenizer results
func (varnam *Varnam) TransliterateGreedyTokenized(word string) []Suggestion {
	ctx := context.Background()

	tokens := varnam.tokenizeWord(ctx, word, VARNAM_MATCH_EXACT, false)
	return varnam.tokensToSuggestions(ctx, tokens, false, varnam.TokenizerSuggestionsLimit)
}

// ReverseTransliterate do a reverse transliteration
func (varnam *Varnam) ReverseTransliterate(word string) ([]Suggestion, error) {
	var results []Suggestion
	ctx := context.Background()

	tokens := varnam.splitTextByConjunct(ctx, word)

	if varnam.Debug {
		fmt.Println(tokens)
	}

	for i := range tokens {
		for j, symbol := range tokens[i].symbols {
			tokens[i].symbols[j].Value1 = symbol.Pattern
			tokens[i].symbols[j].Value2 = symbol.Pattern
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

	err = varnam.InitVST(vstPath)
	if err != nil {
		return nil, err
	}

	// One dictionary for one language, not for different scheme
	dictPath = findLearningsFilePath(varnam.SchemeDetails.LangCode)

	err = varnam.InitDict(dictPath)
	if err != nil {
		return nil, err
	}

	varnam.setDefaultConfig()

	return &varnam, nil
}

// Close close db connections
func (varnam *Varnam) Close() error {
	if varnam.vstConn != nil {
		varnam.vstConn.Close()
	}
	if varnam.dictConn != nil {
		varnam.dictConn.Close()
	}
	return nil
}
