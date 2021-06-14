package govarnam

import (
	"os"
	"path"
)

const VARNAM_MATCH_EXACT = 1
const VARNAM_MATCH_POSSIBILITY = 2
const VARNAM_MATCH_ALL = 3

const VARNAM_TOKEN_CHAR = 1
const VARNAM_TOKEN_SYMBOL = 2

const VARNAM_TOKEN_BASIC_WEIGHT = 10

const VARNAM_SYMBOL_VOWEL = 1
const VARNAM_SYMBOL_CONSONANT = 2
const VARNAM_SYMBOL_DEAD_CONSONANT = 3
const VARNAM_SYMBOL_CONSONANT_VOWEL = 4
const VARNAM_SYMBOL_NUMBER = 5
const VARNAM_SYMBOL_SYMBOL = 6
const VARNAM_SYMBOL_ANUSVARA = 7
const VARNAM_SYMBOL_VISARGA = 8
const VARNAM_SYMBOL_VIRAMA = 9
const VARNAM_SYMBOL_OTHER = 10
const VARNAM_SYMBOL_NON_JOINER = 11
const VARNAM_SYMBOL_JOINER = 12
const VARNAM_SYMBOL_PERIOD = 13

// VARNAM_LEARNT_WORD_MIN_CONFIDENCE Minimum confidence for learnt words
const VARNAM_LEARNT_WORD_MIN_CONFIDENCE = 30

// VARNAM_VST_DIR VST lookiup directories according to priority
var VARNAM_VST_DIR = [2]string{
	// "/usr/share/varnam/vst",
	// "/usr/local/share/varnam/vst",
	"/usr/local/share/varnam/vstDEV"}

func findVSTPath(langCode string) *string {
	for _, loc := range VARNAM_VST_DIR {
		temp := path.Join(loc, langCode+".vst")
		if fileExists(temp) {
			return &temp
		}
	}
	return nil
}

func findLearningsFilePath(langCode string) string {
	var (
		loc string
		dir string
	)

	home := os.Getenv("XDG_DATA_HOME")
	if home == "" {
		home = os.Getenv("HOME")
		dir = path.Join(home, ".local", "share", "varnam", "suggestionsDEV")
	} else {
		dir = path.Join(home, "varnam", "suggestionsDEV")
	}
	loc = path.Join(dir, langCode+".vst.learnings")

	return loc
}
