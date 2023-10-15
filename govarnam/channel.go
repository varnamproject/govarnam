package govarnam

import (
	"context"
	"log"
	"time"
)

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
