package govarnam

func (varnam *Varnam) channelTokensToSuggestions(tokens []Token, greedy bool, partial bool, channel chan []Suggestion) {
	channel <- varnam.tokensToSuggestions(tokens, greedy, partial)
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
