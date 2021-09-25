package govarnamgo

import (
	"context"
	"testing"
)

func TestMLInscriptTransilterateGreedyTokenized(t *testing.T) {
	varnam := getVarnamInstance("ml-inscript")

	assertEqual(t, varnam.TransliterateGreedyTokenized("EnhdhgB")[0].Word, "ആലപ്പുഴ")
}

func TestMLInscriptGetSuggestions(t *testing.T) {
	varnam := getVarnamInstance("ml-inscript")

	varnam.Learn("ഇടുക്കി", 0)

	sugs, err := varnam.GetSuggestions(
		context.Background(),
		varnam.TransliterateGreedyTokenized("F'")[0].Word, // ഇട
	)
	checkError(err)

	assertEqual(t, sugs[0].Word, "ഇടുക്കി")
}
