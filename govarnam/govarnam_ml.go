package govarnam

func (varnam *Varnam) mlPatternWordPartializer(sug *Suggestion) {
	lastChar, size := getLastCharacter(sug.Word)
	if lastChar == "ർ" {
		// റ because english words doesn't have ര sound
		sug.Word = sug.Word[0:len(sug.Word)-size] + "റ"
	} else if lastChar == "ൻ" {
		sug.Word = sug.Word[0:len(sug.Word)-size] + "ന"
	} else if lastChar == "ൽ" {
		sug.Word = sug.Word[0:len(sug.Word)-size] + "ല"
	} else if lastChar == "ൺ" {
		sug.Word = sug.Word[0:len(sug.Word)-size] + "ണ"
	} else if lastChar == "ൾ" {
		sug.Word = sug.Word[0:len(sug.Word)-size] + "ള"
	}
}
