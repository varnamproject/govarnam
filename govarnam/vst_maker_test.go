package govarnam

import (
	"path"
	"testing"
)

func TestCreatTokenWithoutBuffering(t *testing.T) {
	// varnam := getVarnamInstance("ml")
	varnam, err := VMInit(path.Join(testTempDir, "test.vst"))

	checkError(err)

	err = varnam.VMCreateToken("pattern", "value1", "value2", "value3", "tag", VARNAM_SYMBOL_VOWEL, VARNAM_MATCH_EXACT, 0, 0, false)
	checkError(err)
}

func TestCreatTokenWithBuffering(t *testing.T) {
	varnam, err := VMInit(path.Join(testTempDir, "test.vst"))

	checkError(err)
	err = varnam.VMCreateToken("pattern1", "value1", "value2", "", "value3", VARNAM_SYMBOL_VOWEL, VARNAM_MATCH_EXACT, 0, 0, true)
	checkError(err)

	err = varnam.VMFlushBuffer()
	checkError(err)
}

func TestDeleteToken(t *testing.T) {
	varnam, err := VMInit(path.Join(testTempDir, "test.vst"))
	checkError(err)

	symbol := Symbol{Pattern: "pattern"}
	err = varnam.VMDeleteToken(symbol)
	checkError(err)

	err = varnam.VMDeleteToken(Symbol{Pattern: "pattern1"})
	checkError(err)

}

func TestSetSchemeDetails(t *testing.T) {
	varnam, err := VMInit(path.Join(testTempDir, "test.vst"))
	checkError(err)
	sd := SchemeDetails{
		LangCode:    "tl",
		DisplayName: "Test",
		Author:      "Anon",
	}
	err = varnam.VMSetSchemeDetails(sd)
	checkError(err)
}

func TestIgnoreDuplicates(t *testing.T) {
	varnam, err := VMInit(path.Join(testTempDir, "test.vst"))
	checkError(err)

	// disable IgnoreDuplicateTokens flag
	varnam.VSTMakerConfig.IgnoreDuplicateTokens = false

	// should be success
	err = varnam.VMCreateToken("pattern2", "value1", "value2", "value3", "tag", VARNAM_SYMBOL_VOWEL, VARNAM_MATCH_EXACT, 0, 0, false)
	checkError(err)

	// should return error
	err = varnam.VMCreateToken("pattern2", "value1", "value2", "value3", "tag", VARNAM_SYMBOL_VOWEL, VARNAM_MATCH_EXACT, 0, 0, false)
	assertEqual(t, err != nil, true)

	// enable IgnoreDuplicateTokens flag
	varnam.VSTMakerConfig.IgnoreDuplicateTokens = true

	err = varnam.VMCreateToken("pattern2", "value1", "value2", "value3", "tag", VARNAM_SYMBOL_VOWEL, VARNAM_MATCH_EXACT, 0, 0, false)
	checkError(err)
}
