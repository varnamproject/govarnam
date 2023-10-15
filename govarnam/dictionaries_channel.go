package govarnam

import (
	"context"
	"fmt"
	"log"
	"time"
)

func (varnam *Varnam) channelGetFromDictionaries(ctx context.Context, word string, tokens *[]Token, channel chan channelDictionaryResult) {
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
		for _, dictConfig := range varnam.DictsConfig {
			varnam.getSuggestionsFromDict()
		}

		channel <- channelDictionaryResult{
			exactWords,
			exactMatches,
			moreSuggestions,
		}
		close(channel)
	}
}

func (varnam *Varnam) channelGetFromPatternDictionaries(ctx context.Context, word string, channel chan channelDictionaryResult) {
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

func (varnam *Varnam) channelGetMoreFromDictionaries(ctx context.Context, sugs []Suggestion, channel chan MoreDictionaryResult) {
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
