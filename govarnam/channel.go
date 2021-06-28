package govarnam

func (varnam *Varnam) channelTokensToGreedySuggestions(tokens []Token, channel chan []Suggestion) {
	// Altering tokens directly will affect others
	tokensCopy := make([]Token, len(tokens))
	copy(tokensCopy, tokens)

	tokensCopy = removeNonExactTokens(tokensCopy)
	channel <- varnam.tokensToSuggestions(tokensCopy, false)

	close(channel)
}

func (varnam *Varnam) channelGetFromDictionary(tokens []Token, channel chan DictionaryResult) {
	channel <- varnam.getFromDictionary(tokens)
	close(channel)
}

func (varnam *Varnam) channelGetFromPatternDictionary(word string, channel chan []PatternDictionarySuggestion) {
	channel <- varnam.getFromPatternDictionary(word)
	close(channel)
}

func (varnam *Varnam) channelGetMoreFromDictionary(sugs []Suggestion, channel chan [][]Suggestion) {
	channel <- varnam.getMoreFromDictionary(sugs)
	close(channel)
}
