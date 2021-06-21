package govarnam

import "unicode/utf8"

func getFirstCharacter(input string) (string, int) {
	r, size := utf8.DecodeRuneInString(input)
	if r == utf8.RuneError && (size == 0 || size == 1) {
		size = 0
	}
	return input[0:size], size
}

func getLastCharacter(input string) (string, int) {
	r, size := utf8.DecodeLastRuneInString(input)
	if r == utf8.RuneError && (size == 0 || size == 1) {
		size = 0
	}
	return input[len(input)-size:], size
}

func (varnam *Varnam) removeLastVirama(input string) string {
	char, size := getLastCharacter(input)
	if char == varnam.LangRules.Virama {
		return input[0 : len(input)-size]
	}
	return input
}
