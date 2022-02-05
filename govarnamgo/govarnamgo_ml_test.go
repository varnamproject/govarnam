package govarnamgo

import (
	"context"
	"os"
	"path"
	"testing"
)

func TestSchemeDetails(t *testing.T) {
	varnam := getVarnamInstance("ml")

	assertEqual(t, varnam.GetSchemeDetails().Identifier, "ml")
	assertEqual(t, varnam.GetSchemeDetails().DisplayName, "Malayalam")
}

func TestTransliterate(t *testing.T) {
	varnam := getVarnamInstance("ml")

	result, err := varnam.Transliterate(context.Background(), "nithyam")
	checkError(err)

	assertEqual(t, result[0].Word, "നിത്യം")
}

func TestTransliterateAdvanced(t *testing.T) {
	varnam := getVarnamInstance("ml")

	result, err := varnam.TransliterateAdvanced(context.Background(), "nithyam")
	checkError(err)

	assertEqual(t, result.TokenizerSuggestions[0].Word, "നിത്യം")
}

func TestReverseTransliterate(t *testing.T) {
	varnam := getVarnamInstance("ml")

	rt, err := varnam.ReverseTransliterate("നിത്യം")
	checkError(err)

	assertEqual(t, rt[0].Word, "nithyam")
}

func TestLearn(t *testing.T) {
	varnam := getVarnamInstance("ml")

	filePath := path.Join(testTempDir, "report.txt")

	file, err := os.Create(filePath)
	if err != nil {
		return
	}
	defer file.Close()

	file.WriteString(`നിത്യഹരിത 120
	വൃക്ഷമാണ് 89
	ഒരേയൊരു 45
	ഏഷ്യയുടെ 100
	മേലാപ്പും 12
	aadc 10`)

	learnStatus, verr := varnam.LearnFromFile(filePath)
	checkError(verr)

	assertEqual(t, learnStatus.TotalWords, 6)
	assertEqual(t, learnStatus.FailedWords, 1)

	result, err := varnam.TransliterateAdvanced(context.Background(), "nithyaharitha")
	assertEqual(t, result.ExactWords[0].Weight, 120)
	result, err = varnam.TransliterateAdvanced(context.Background(), "melaappum")
	assertEqual(t, result.ExactWords[0].Weight, 12)
}

func TestRecentlyLearnedWords(t *testing.T) {
	varnam := getVarnamInstance("ml")

	words := []string{"ആലപ്പുഴ", "എറണാകുളം", "തൃശ്ശൂർ", "മലപ്പുറം", "കോഴിക്കോട്"}
	for _, word := range words {
		varnam.Learn(word, 0)
	}

	result, err := varnam.GetRecentlyLearntWords(context.Background(), 0, len(words))
	checkError(err)

	assertEqual(t, len(result), len(words))
	for i, sug := range result {
		assertEqual(t, sug.Word, words[len(words)-i-1])
	}
}

func TestSearchSymbolTable(t *testing.T) {
	varnam := getVarnamInstance("ml")

	symbol := NewSearchSymbol()
	symbol.Pattern = "la"
	result := varnam.SearchSymbolTable(context.Background(), symbol)

	assertEqual(t, result[0].Value1, "ല")
}
