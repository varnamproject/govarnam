package govarnam

import "unicode"

/**
 * govarnam - An Indian language transliteration library
 * Copyright Subin Siby <mail at subinsb (.) com>, 2021
 * Licensed under AGPL-3.0-only. See LICENSE.txt
 */

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
	} else if lastChar == "ം" {
		sug.Word = sug.Word[0:len(sug.Word)-size] + "മ"
	}
}

func (varnam *Varnam) getUnicodeBlock() unicode.RangeTable {
	switch varnam.SchemeDetails.LangCode {
	case "kn":
		return unicode.RangeTable{R16: []unicode.Range16{{0x0C80, 0x0CFF, 1}}}
	case "ml":
		return unicode.RangeTable{R16: []unicode.Range16{{0x0D00, 0x0D7F, 1}}}
	default:
		return unicode.RangeTable{}
	}
	// TODO add for all languages
}
