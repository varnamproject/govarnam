package main

/**
 * govarnam - An Indian language transliteration library
 * Copyright Subin Siby, 2021
 * Licensed under AGPL-3.0-only
 */

// #cgo LDFLAGS: -L${SRCDIR}/../ -lgovarnam
// #cgo CFLAGS: -I${SRCDIR}/../ -DHAVE_SNPRINTF -DPREFER_PORTABLE_SNPRINTF -DNEED_ASPRINTF
// #include <libgovarnam.h>
import "C"

import (
	"flag"
	"fmt"
	"log"
	"os"
)

// Suggestion suggestion
type Suggestion struct {
	Word      string
	Weight    int
	LearnedOn int
}

// TransliterationResult result
type TransliterationResult struct {
	ExactMatch            []Suggestion
	Suggestions           []Suggestion
	GreedyTokenized       []Suggestion
	DictionaryResultCount int
}

// Convert a C Suggestion to Go
func makeSuggestion(cSug C.struct_Suggestion_t) Suggestion {
	var sug Suggestion
	sug.Word = C.GoString(cSug.Word)
	sug.Weight = int(cSug.Weight)
	sug.LearnedOn = int(cSug.LearnedOn)
	return sug
}

func main() {
	debugFlag := flag.Bool("debug", false, "Enable debugging outputs")
	langFlag := flag.String("lang", "", "Language")

	learnFlag := flag.Bool("learn", false, "Learn a word")
	unlearnFlag := flag.Bool("unlearn", false, "Unlearn a word")
	trainFlag := flag.Bool("train", false, "Train a word with a particular pattern. 2 Arguments: Pattern & Word")

	learnFromFileFlag := flag.Bool("learn-from-file", false, "Learn words in a file")

	indicDigitsFlag := flag.Bool("digits", false, "Use indic digits")
	greedy := flag.Bool("greedy", false, "Show only exactly matched suggestions")

	flag.Parse()

	err := C.varnam_init_from_id(C.CString(*langFlag))

	if err != C.VARNAM_SUCCESS {
		log.Fatal(err)
		return
	}

	if *debugFlag {
		C.varnam_debug(C.int(1))
	}
	if *indicDigitsFlag {
		C.varnam_set_indic_digits(C.int(1))
	}

	args := flag.Args()

	if *trainFlag {
		pattern := args[0]
		word := args[1]

		if C.varnam_train(C.CString(pattern), C.CString(word)) == C.VARNAM_SUCCESS {
			fmt.Printf("Trained %s => %s\n", pattern, word)
		} else {
			fmt.Printf(C.GoString(C.varnam_get_last_error()) + "\n")
		}
	} else if *learnFlag {
		word := args[0]

		if C.varnam_learn(C.CString(word), 0) == C.VARNAM_SUCCESS {
			fmt.Printf("Learnt %s\n", word)
		} else {
			fmt.Printf("Couldn't learn %s", word)
		}
	} else if *unlearnFlag {
		word := args[0]

		if C.varnam_unlearn(C.CString(word)) == C.VARNAM_SUCCESS {
			fmt.Printf("Unlearnt %s\n", word)
		} else {
			fmt.Printf("Couldn't learn %s", word)
		}
	} else if *learnFromFileFlag {
		file, err := os.Open(args[0])
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()

		// C.LearnFromFile(file)
	} else {
		var cResults *C.TransliterationResult

		if *greedy {
			// results = C.TransliterateGreedy(args[0])
		} else {
			cResults = C.varnam_transliterate(C.CString(args[0]))
		}

		var i int
		var results TransliterationResult

		var exactMatch []Suggestion
		i = 0
		for i < int(C.varray_length(cResults.ExactMatch)) {
			cSug := *(*C.Suggestion)(C.varray_get(cResults.ExactMatch, C.int(i)))
			sug := makeSuggestion(cSug)
			exactMatch = append(exactMatch, sug)
			i++
		}
		results.ExactMatch = exactMatch

		var suggestions []Suggestion
		i = 0
		for i < int(C.varray_length(cResults.Suggestions)) {
			cSug := *(*C.Suggestion)(C.varray_get(cResults.Suggestions, C.int(i)))
			sug := makeSuggestion(cSug)
			suggestions = append(suggestions, sug)
			i++
		}
		results.Suggestions = suggestions

		var greedyTokenized []Suggestion
		i = 0
		for i < int(C.varray_length(cResults.GreedyTokenized)) {
			cSug := *(*C.Suggestion)(C.varray_get(cResults.Suggestions, C.int(i)))
			sug := makeSuggestion(cSug)
			greedyTokenized = append(greedyTokenized, sug)
			i++
		}
		results.GreedyTokenized = greedyTokenized

		if len(results.ExactMatch) > 0 {
			fmt.Println("Exact Matches")
			for _, sug := range results.ExactMatch {
				fmt.Println(sug.Word + " " + fmt.Sprint(sug.Weight))
			}
		}
		if len(results.Suggestions) > 0 {
			fmt.Println("Suggestions")
			for _, sug := range results.Suggestions {
				fmt.Println(sug.Word + " " + fmt.Sprint(sug.Weight))
			}
		}
		fmt.Println("Greedy Tokenized")
		for _, sug := range results.GreedyTokenized {
			fmt.Println(sug.Word + " " + fmt.Sprint(sug.Weight))
		}
	}
}
