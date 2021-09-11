package govarnam

import (
	"fmt"
	"os"
	"path"
)

/* General */
const ZWNJ = "\u200c"
const ZWJ = "\u200d"

/* Pattern matching */
const VARNAM_MATCH_EXACT = 1
const VARNAM_MATCH_POSSIBILITY = 2
const VARNAM_MATCH_ALL = 3

/* Type of tokens */
const VARNAM_TOKEN_CHAR = 1   // Non-lang characters like A, B, 1, * etc.
const VARNAM_TOKEN_SYMBOL = 2 // Lang characters

/* A symbol token's maximum possible weight value */
const VARNAM_TOKEN_BASIC_WEIGHT = 10

/* Available type of symbol tokens */
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

/* Token acceptance rules */
const VARNAM_TOKEN_ACCEPT_ALL = 0
const VARNAM_TOKEN_ACCEPT_IF_STARTS_WITH = 1
const VARNAM_TOKEN_ACCEPT_IF_IN_BETWEEN = 2
const VARNAM_TOKEN_ACCEPT_IF_ENDS_WITH = 3

// VARNAM_LEARNT_WORD_MIN_WEIGHT Minimum weight/confidence for learnt words.
const VARNAM_LEARNT_WORD_MIN_WEIGHT = 30

const CHIL_TAG = "chill"

// VARNAM_VST_DIR VST lookiup directories according to priority
func getVSTLookupDirs() []string {
	return []string{
		// libvarnam used to use "vst" folder
		os.Getenv("VARNAM_VST_DIR"),
		"schemes",
		"/usr/local/share/varnam/schemes",
		"/usr/share/varnam/schemes",
	}
}

//FindVSTDir Get the VST storing directory
func FindVSTDir() (string, error) {
	for _, loc := range getVSTLookupDirs() {
		if dirExists(loc) {
			return loc, nil
		}
	}
	return "", fmt.Errorf("Couldn't find VST directory")
}

func findVSTPath(schemeID string) (string, error) {
	for _, dir := range getVSTLookupDirs() {
		temp := path.Join(dir, schemeID+".vst")
		if fileExists(temp) {
			return temp, nil
		}
	}
	return "", fmt.Errorf("Couldn't find VST for %q", schemeID)
}

func findLearningsFilePath(langCode string) string {
	var (
		loc string
		dir string
	)

	// libvarnam used to use "suggestions" folder

	learningsDir := os.Getenv("VARNAM_LEARNINGS_DIR")
	if learningsDir != "" {
		dir = learningsDir
	} else {
		home := os.Getenv("XDG_DATA_HOME")
		if home != "" {
			dir = path.Join(home, "varnam", "learnings")
		} else {
			home = os.Getenv("HOME")
			dir = path.Join(home, ".local", "share", "varnam", "learnings")
		}
	}

	loc = path.Join(dir, langCode+".vst.learnings")

	return loc
}

var LOG_TIME_TAKEN = os.Getenv("GOVARNAM_LOG_TIME_TAKEN") != ""
