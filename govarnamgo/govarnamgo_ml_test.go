package govarnamgo

import (
	"context"
	"os"
	"path"
	"testing"
)

func TestTransliterate(t *testing.T) {
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

	assertEqual(t, varnam.Transliterate(context.Background(), "nithyaharitha").ExactMatches[0].Weight, 120)
	assertEqual(t, varnam.Transliterate(context.Background(), "melaappum").ExactMatches[0].Weight, 12)
}

func TestSearchSymbolTable(t *testing.T) {
	varnam := getVarnamInstance("ml")

	var symbol Symbol
	symbol.Pattern = "la"
	result := varnam.SearchSymbolTable(context.Background(), symbol)

	assertEqual(t, result[0].Value1, "ല")
}
