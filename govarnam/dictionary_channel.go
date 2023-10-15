package govarnam

import (
	"context"
	sql "database/sql"
	"fmt"
	"log"
	"time"
)

type DictionarySuggestions struct {
	exactWords   []Suggestion
	exactMatches []Suggestion
	suggestions  []Suggestion
}

func (varnam *Varnam) getSuggestionsFromDictionary(
	ctx context.Context,
	dbConn *sql.DB,
	word string,
	tokens *[]Token,
) DictionarySuggestions {
	var (
		exactWords      []Suggestion
		exactMatches    []Suggestion
		moreSuggestions []Suggestion
	)

	start := time.Now()

	dictResult := varnam.getFromDictionary(ctx, dbConn, tokens)

	if varnam.Debug {
		fmt.Println("Dictionary results:", dictResult)
	}

	if len(dictResult.exactMatches) > 0 {
		start := time.Now()

		// Exact words can be determined finally
		// with help of this function's result
		moreFromDict := varnam.getMoreFromDictionary(ctx, dbConn, dictResult.exactMatches)

		if varnam.Debug {
			fmt.Println("More dictionary results:", moreFromDict)
		}

		// dictResult.exactMatches will have both matches and exact words.
		// getMoreFromDictionary() will separate out the exact words.
		exactWords = moreFromDict.exactWords

		// Intersection of slices.
		// exactMatches shouldn't have items from exactWords
		hash := make(map[string]bool)
		for i := range exactWords {
			hash[exactWords[i].Word] = true
		}
		for _, sug := range dictResult.exactMatches {
			if _, found := hash[sug.Word]; !found {
				exactMatches = append(exactMatches, sug)
			}
		}

		for _, sugSet := range moreFromDict.moreSuggestions {
			moreSuggestions = append(moreSuggestions, sugSet...)
		}

		if LOG_TIME_TAKEN {
			log.Printf("%s took %v\n", "getMoreFromDictionary", time.Since(start))
		}
	}

	if len(dictResult.partialMatches) > 0 {
		// Tokenize the word after the longest match found in dictionary
		restOfWord := word[dictResult.longestMatchPosition+1:]

		start := time.Now()

		moreSuggestions = varnam.tokenizeRestOfWord(
			ctx,
			restOfWord,
			dictResult.partialMatches,
			varnam.DictionarySuggestionsLimit,
		)

		if LOG_TIME_TAKEN {
			log.Printf("%s took %v\n", "tokenizeRestOfWord", time.Since(start))
		}
	}

	if LOG_TIME_TAKEN {
		log.Printf("%s took %v\n", "channelGetFromDictionary", time.Since(start))
	}

	return DictionarySuggestions{
		exactWords,
		exactMatches,
		moreSuggestions,
	}
}

func (varnam *Varnam) channelGetFromPatternDictionary(
	ctx context.Context,
	dbConn *sql.DB,
	word string,
	channel chan channelDictionaryResult,
) {
	var (
		exactWords      []Suggestion
		moreSuggestions []Suggestion
	)

	select {
	case <-ctx.Done():
		close(channel)
		return
	default:
		start := time.Now()

		patternDictSugs := varnam.getFromPatternDictionary(ctx, dbConn, word)

		if len(patternDictSugs) > 0 {
			if varnam.Debug {
				fmt.Println("Pattern dictionary results:", patternDictSugs)
			}

			var partialMatches []PatternDictionarySuggestion

			for _, match := range patternDictSugs {
				if match.Length < len(word) {
					sug := &match.Sug

					// Increase weight on length matched.
					// 50 because half of 100%
					sug.Weight += match.Length * 50

					for _, cb := range varnam.PatternWordPartializers {
						cb(sug)
					}

					partialMatches = append(partialMatches, match)
				} else if match.Length == len(word) {
					// Same length, exact word matched
					exactWords = append(exactWords, match.Sug)
				} else {
					moreSuggestions = append(moreSuggestions, match.Sug)
				}
			}

			perMatchLimit := varnam.PatternDictionarySuggestionsLimit

			if len(partialMatches) > 0 && perMatchLimit > len(partialMatches) {
				perMatchLimit = perMatchLimit / len(partialMatches)
			}

			for i := range partialMatches {
				restOfWord := word[partialMatches[i].Length:]

				filled := varnam.tokenizeRestOfWord(
					ctx,
					restOfWord,
					[]Suggestion{partialMatches[i].Sug},
					perMatchLimit,
				)

				moreSuggestions = append(moreSuggestions, filled...)

				if len(moreSuggestions) >= varnam.PatternDictionarySuggestionsLimit {
					break
				}
			}
		}

		if LOG_TIME_TAKEN {
			log.Printf("%s took %v\n", "channelGetFromPatternDictionary", time.Since(start))
		}

		channel <- channelDictionaryResult{
			exactWords,
			[]Suggestion{}, // Not applicable for patterns dictionary
			moreSuggestions,
		}
		close(channel)
	}
}

func (varnam *Varnam) channelGetMoreFromDictionary(
	ctx context.Context,
	dbConn *sql.DB,
	sugs []Suggestion,
	channel chan MoreDictionaryResult,
) {
	select {
	case <-ctx.Done():
		close(channel)
		return
	default:
		start := time.Now()

		result := varnam.getMoreFromDictionary(ctx, dbConn, sugs)

		if LOG_TIME_TAKEN {
			log.Printf("%s took %v\n", "channelGetMoreFromDictionary", time.Since(start))
		}

		channel <- result
		close(channel)
	}
}
