package govarnam

import (
	"context"
	"fmt"
)

type channelDictionaryResult struct {
	exactMatches []Suggestion
	suggestions  []Suggestion
}

func (varnam *Varnam) channelTokensToGreedySuggestions(ctx context.Context, tokens []Token, channel chan []Suggestion) {
	// Altering tokens directly will affect others
	tokensCopy := make([]Token, len(tokens))
	copy(tokensCopy, tokens)

	tokensCopy = removeNonExactTokens(tokensCopy)
	channel <- varnam.tokensToSuggestions(ctx, tokensCopy, false)

	close(channel)
}

func (varnam *Varnam) channelGetFromDictionary(ctx context.Context, word string, tokens []Token, channel chan channelDictionaryResult) {
	var (
		dictResults  []Suggestion
		exactMatches []Suggestion
	)

	dictSugs := varnam.getFromDictionary(ctx, tokens)

	if varnam.Debug {
		fmt.Println("Dictionary results:", dictSugs)
	}

	if len(dictSugs.sugs) > 0 {
		if dictSugs.exactMatch == false {
			// These will be partial words
			restOfWord := word[dictSugs.longestMatchPosition+1:]
			dictResults = varnam.tokenizeRestOfWord(ctx, restOfWord, dictSugs.sugs)
		} else {
			exactMatches = dictSugs.sugs

			// Since partial words are in dictionary, exactMatch will be TRUE
			// for pathway to a word. Hence we're calling this here
			moreFromDict := varnam.getMoreFromDictionary(ctx, dictSugs.sugs)

			if varnam.Debug {
				fmt.Println("More dictionary results:", moreFromDict)
			}

			for _, sugSet := range moreFromDict {
				dictResults = append(dictResults, sugSet...)
			}
		}
	}

	channel <- channelDictionaryResult{exactMatches, dictResults}

	close(channel)
}

func (varnam *Varnam) channelGetFromPatternDictionary(ctx context.Context, word string, channel chan channelDictionaryResult) {
	var (
		dictResults  []Suggestion
		exactMatches []Suggestion
	)

	patternDictSugs := varnam.getFromPatternDictionary(ctx, word)

	if len(patternDictSugs) > 0 {
		if varnam.Debug {
			fmt.Println("Pattern dictionary results:", patternDictSugs)
		}

		for _, match := range patternDictSugs {
			if match.Length < len(word) {
				restOfWord := word[match.Length:]
				filled := varnam.tokenizeRestOfWord(ctx, restOfWord, []Suggestion{match.Sug})
				dictResults = append(dictResults, filled...)
			} else if match.Length == len(word) {
				// Same length
				exactMatches = append(exactMatches, match.Sug)
			} else {
				dictResults = append(dictResults, match.Sug)
			}
		}
	}

	channel <- channelDictionaryResult{exactMatches, dictResults}

	close(channel)
}

func (varnam *Varnam) channelGetMoreFromDictionary(ctx context.Context, sugs []Suggestion, channel chan [][]Suggestion) {
	channel <- varnam.getMoreFromDictionary(ctx, sugs)
	close(channel)
}
