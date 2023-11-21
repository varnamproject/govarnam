package govarnam

import (
	"context"
	"fmt"
	"log"
	"time"
)

type channelDictionaryResult struct {
	exactWords   []Suggestion
	exactMatches []Suggestion
	suggestions  []Suggestion
}

func (varnam *Varnam) channelTokenizeWord(ctx context.Context, word string, matchType int, partial bool, channel chan *[]Token) {
	select {
	case <-ctx.Done():
		close(channel)
		return
	default:
		start := time.Now()

		tokens := varnam.tokenizeWord(ctx, word, matchType, partial)

		if LOG_TIME_TAKEN {
			log.Printf("%s took %v\n", "channelTokenizeWord", time.Since(start))
		}

		channel <- tokens
		close(channel)
	}
}

func (varnam *Varnam) channelTokensToSuggestions(ctx context.Context, tokens *[]Token, limit int, channel chan []Suggestion) {
	select {
	case <-ctx.Done():
		close(channel)
		return
	default:
		start := time.Now()

		sugs := varnam.tokensToSuggestions(ctx, tokens, false, limit)

		if LOG_TIME_TAKEN {
			log.Printf("%s took %v\n", "channelTokensToSuggestions", time.Since(start))
		}

		channel <- sugs
		close(channel)
	}
}

func (varnam *Varnam) channelTokensToGreedySuggestions(ctx context.Context, tokens *[]Token, channel chan []Suggestion) {
	select {
	case <-ctx.Done():
		close(channel)
		return
	default:
		start := time.Now()

		sugs := varnam.tokensToSuggestions(ctx, tokens, false, varnam.TokenizerSuggestionsLimit)

		if LOG_TIME_TAKEN {
			log.Printf("%s took %v\n", "channelTokensToGreedySuggestions", time.Since(start))
		}

		channel <- sugs
		close(channel)
	}
}

func (varnam *Varnam) channelGetFromDictionary(ctx context.Context, word string, tokens *[]Token, channel chan channelDictionaryResult) {
	var (
		exactWords      []Suggestion
		exactMatches    []Suggestion
		moreSuggestions []Suggestion
	)

	select {
	case <-ctx.Done():
		close(channel)
		return
	default:
		start := time.Now()

		dictResult := varnam.getFromDictionary(ctx, tokens)

		if varnam.Debug {
			fmt.Println("Dictionary results:", dictResult)
		}

		if len(dictResult.exactMatches) > 0 {
			start := time.Now()

			// Exact words can be determined finally
			// with help of this function's result
			moreFromDict := varnam.getMoreFromDictionary(ctx, dictResult.exactMatches)

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
			restOfWord := string([]rune(word)[dictResult.longestMatchPosition+1:])

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

		channel <- channelDictionaryResult{
			exactWords,
			exactMatches,
			moreSuggestions,
		}
		close(channel)
	}
}

func (varnam *Varnam) channelGetFromPatternDictionary(ctx context.Context, word string, channel chan channelDictionaryResult) {
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

		patternDictSugs := varnam.getFromPatternDictionary(ctx, word)

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

func (varnam *Varnam) channelGetMoreFromDictionary(ctx context.Context, sugs []Suggestion, channel chan MoreDictionaryResult) {
	select {
	case <-ctx.Done():
		close(channel)
		return
	default:
		start := time.Now()

		result := varnam.getMoreFromDictionary(ctx, sugs)

		if LOG_TIME_TAKEN {
			log.Printf("%s took %v\n", "channelGetMoreFromDictionary", time.Since(start))
		}

		channel <- result
		close(channel)
	}
}
