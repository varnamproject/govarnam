package govarnam

import (
	"context"
	"path"
	"strings"
	"testing"
)

// initialize VST Maker for tests
func initTestVM() (*Varnam, error) {
	return VMInit(path.Join(testTempDir, "test.vst"))
}

func TestCreatTokenWithoutBuffering(t *testing.T) {
	varnam, err := initTestVM()
	checkError(err)

	err = varnam.VMCreateToken("pattern", "value1", "value2", "value3", "tag", VARNAM_SYMBOL_VOWEL, VARNAM_MATCH_EXACT, 0, 0, false)
	checkError(err)

	search := NewSearchSymbol()
	search.Pattern = "pattern"
	symbols, err := varnam.SearchSymbolTable(context.Background(), search)
	checkError(err)
	assertEqual(t, len(symbols) == 1, true)
}

func TestCreatTokenWithBuffering(t *testing.T) {
	varnam, err := initTestVM()

	checkError(err)
	err = varnam.VMCreateToken("pattern1", "value1", "value2", "", "value3", VARNAM_SYMBOL_VOWEL, VARNAM_MATCH_EXACT, 0, 0, true)
	checkError(err)

	search := NewSearchSymbol()
	search.Pattern = "pattern1"
	symbols, err := varnam.SearchSymbolTable(context.Background(), search)
	checkError(err)
	assertEqual(t, len(symbols) == 1, true)

	err = varnam.VMFlushBuffer()
	checkError(err)
}

func TestDeleteToken(t *testing.T) {
	varnam, err := initTestVM()
	checkError(err)

	search := NewSearchSymbol()
	search.Pattern = "pattern"
	err = varnam.VMDeleteToken(search)
	checkError(err)

	symbols, err := varnam.SearchSymbolTable(context.Background(), search)
	checkError(err)
	assertEqual(t, len(symbols) == 0, true)

	search.Pattern = "pattern1"
	err = varnam.VMDeleteToken(search)
	checkError(err)

	symbols, err = varnam.SearchSymbolTable(context.Background(), search)
	checkError(err)
	assertEqual(t, len(symbols) == 0, true)
}

func TestGetAllTokens(t *testing.T) {
	varnam, err := initTestVM()
	checkError(err)

	err = varnam.VMCreateToken("pattern", "value1", "value2", "", "", VARNAM_SYMBOL_VOWEL, VARNAM_MATCH_EXACT, 0, 0, false)
	checkError(err)

	err = varnam.VMCreateToken("pattern1", "value11", "value21", "", "", VARNAM_SYMBOL_VOWEL, VARNAM_MATCH_EXACT, 0, 0, false)
	checkError(err)

	err = varnam.VMCreateToken("pattern2", "value12", "value22", "", "", VARNAM_SYMBOL_VOWEL, VARNAM_MATCH_EXACT, 0, 0, false)
	checkError(err)

	varnam.VSTMakerConfig.UseDeadConsonants = false

	err = varnam.VMCreateToken("r", "v", "v", "", "", VARNAM_SYMBOL_CONSONANT, VARNAM_MATCH_EXACT, 0, 0, false)
	checkError(err)

	err = varnam.VMCreateToken("r", "v12", "v", "", "", VARNAM_SYMBOL_CONSONANT, VARNAM_MATCH_POSSIBILITY, 0, 0, false)
	checkError(err)

	search := NewSearchSymbol()
	search.Type = VARNAM_SYMBOL_VOWEL

	symbols, err := varnam.SearchSymbolTable(context.Background(), search)
	checkError(err)
	assertEqual(t, len(symbols) == 3, true)

	search.Type = VARNAM_SYMBOL_CONSONANT
	symbols, err = varnam.SearchSymbolTable(context.Background(), search)
	checkError(err)
	assertEqual(t, len(symbols) == 2, true)
}

func TestSetSchemeDetails(t *testing.T) {
	varnam, err := initTestVM()
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
	varnam, err := initTestVM()
	checkError(err)

	varnam.VSTMakerConfig.IgnoreDuplicateTokens = false

	// should succeed
	err = varnam.VMCreateToken("pattern21", "value1", "value2", "value3", "tag", VARNAM_SYMBOL_VOWEL, VARNAM_MATCH_EXACT, 0, 0, false)
	checkError(err)

	// should return error
	err = varnam.VMCreateToken("pattern21", "value1", "value2", "value3", "tag", VARNAM_SYMBOL_VOWEL, VARNAM_MATCH_EXACT, 0, 0, false)
	assertEqual(t, err != nil, true)

	// Ignore Duplicate Tokens
	varnam.VSTMakerConfig.IgnoreDuplicateTokens = true

	err = varnam.VMCreateToken("pattern21", "value1", "value2", "value3", "tag", VARNAM_SYMBOL_VOWEL, VARNAM_MATCH_EXACT, 0, 0, false)
	checkError(err)
}

func TestCreateDeadConsonants(t *testing.T) {
	varnam, err := initTestVM()
	checkError(err)

	// Use dead consonants
	varnam.VSTMakerConfig.UseDeadConsonants = true

	err = varnam.VMCreateToken("~", "്", "", "tag", "", VARNAM_SYMBOL_VIRAMA, VARNAM_MATCH_EXACT, 1, 0, false)
	checkError(err)

	err = varnam.VMCreateToken("ka", "ക", "", "tag", "", VARNAM_SYMBOL_CONSONANT, VARNAM_MATCH_EXACT, 1, 0, false)
	checkError(err)

	err = varnam.VMCreateToken("p", "പ്", "", "tag", "value3", VARNAM_SYMBOL_CONSONANT, VARNAM_MATCH_EXACT, 1, 0, false)
	checkError(err)

	varnam.VMFlushBuffer()

	search := NewSearchSymbol()
	search.Type = VARNAM_SYMBOL_DEAD_CONSONANT

	symbols, err := varnam.SearchSymbolTable(context.Background(), search)
	checkError(err)

	deadConsonantFound := false
	for _, sym := range symbols {
		if sym.Pattern == "k" {
			deadConsonantFound = true
		}
	}
	assertEqual(t, deadConsonantFound, true)
}

func TestCreateExactMatchDuplicates(t *testing.T) {
	varnam, err := initTestVM()
	checkError(err)

	// should be success
	err = varnam.VMCreateToken("patterna", "value1", "value2", "value3", "tag", VARNAM_SYMBOL_VOWEL, VARNAM_MATCH_EXACT, 0, 0, false)
	checkError(err)

	// disable IgnoreDuplicateTokens flag
	varnam.VSTMakerConfig.IgnoreDuplicateTokens = false

	// should return error
	err = varnam.VMCreateToken("patterna", "value1", "value2", "value3", "tag", VARNAM_SYMBOL_VOWEL, VARNAM_MATCH_EXACT, 0, 0, false)
	assertEqual(t, err != nil, true)
}

func TestCreatePossibilityMatchDuplicates(t *testing.T) {
	varnam, err := initTestVM()
	checkError(err)

	// disable IgnoreDuplicateTokens flag
	varnam.VSTMakerConfig.IgnoreDuplicateTokens = false

	err = varnam.VMCreateToken("pattern3", "value1", "value2", "", "value3", VARNAM_SYMBOL_VOWEL, VARNAM_MATCH_POSSIBILITY, 0, 0, false)
	checkError(err)

	// should be allowed since it has different values
	err = varnam.VMCreateToken("pattern3", "value11", "value22", "", "value3", VARNAM_SYMBOL_VOWEL, VARNAM_MATCH_POSSIBILITY, 0, 0, false)
	checkError(err)

	// should return error
	err = varnam.VMCreateToken("pattern3", "value1", "value2", "", "value3", VARNAM_SYMBOL_VOWEL, VARNAM_MATCH_POSSIBILITY, 0, 0, false)
	assertEqual(t, err != nil, true)
}

func TestRestrictInvalidMatchTypes(t *testing.T) {
	varnam, err := initTestVM()
	checkError(err)

	// should return error
	err = varnam.VMCreateToken("pattern0", "value1", "value2", "", "value3", VARNAM_SYMBOL_VOWEL, 11, 0, 0, false)
	assertEqual(t, err != nil, true)
}

func TestMaxLengthCheck(t *testing.T) {
	varnam, err := initTestVM()
	checkError(err)

	pattern := strings.Builder{}
	pattern.Grow(VARNAM_SYMBOL_MAX + 2)

	value1 := strings.Builder{}
	value1.Grow(VARNAM_SYMBOL_MAX + 2)

	value2 := strings.Builder{}
	value2.Grow(VARNAM_SYMBOL_MAX + 2)

	// should return error
	err = varnam.VMCreateToken(pattern.String(), value1.String(), value2.String(), "", "value3", VARNAM_SYMBOL_VOWEL, VARNAM_MATCH_EXACT, 0, 0, false)
	assertEqual(t, err != nil, true)
}

// TODO: incomplete API
func TestPrefixTree(t *testing.T) {
	// varnam, err := initTestVM()
	// checkError(err)
}
