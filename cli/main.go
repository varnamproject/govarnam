package main

/**
 * govarnam - An Indian language transliteration library
 * Copyright Subin Siby, 2021
 * Licensed under AGPL-3.0-only
 */

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"gitlab.com/subins2000/govarnam/govarnamgo"
)

var varnam *govarnamgo.VarnamHandle

func logVarnamError() {
	log.Fatal(varnam.GetLastError())
}

func main() {
	debugFlag := flag.Bool("debug", false, "Enable debugging outputs")
	schemeFlag := flag.String("s", "", "Scheme ID")

	learnFlag := flag.Bool("learn", false, "Learn a word")
	unlearnFlag := flag.Bool("unlearn", false, "Unlearn a word")
	trainFlag := flag.Bool("train", false, "Train a word with a particular pattern. 2 Arguments: Pattern & Word")

	learnFromFileFlag := flag.Bool("learn-from-file", false, "Learn words in a file")
	trainFromFileFlag := flag.Bool("train-from-file", false, "Train pattern => word from a file.")

	exportFlag := flag.Bool("export", false, "Export learnings to file")
	importFlag := flag.Bool("import", false, "Import learnings from file")

	indicDigitsFlag := flag.Bool("digits", false, "Use indic digits")
	greedy := flag.Bool("greedy", false, "Show only exactly matched suggestions")

	reverseTransliterate := flag.Bool("reverse", false, "Reverse transliterate. Find which pattern to use for a specific word")

	flag.Parse()

	if *schemeFlag == "" {
		log.Fatal("Specifiy a scheme ID with -s")
	}

	var err error
	varnam, err = govarnamgo.InitFromID(*schemeFlag)
	if err != nil {
		log.Fatal(err)
	}

	varnam.Debug(*debugFlag)

	config := govarnamgo.Config{IndicDigits: *indicDigitsFlag, DictionarySuggestionsLimit: 10, TokenizerSuggestionsLimit: 10, TokenizerSuggestionsAlways: true}
	varnam.SetConfig(config)

	args := flag.Args()

	if *trainFlag {
		pattern := args[0]
		word := args[1]

		if varnam.Train(pattern, word) {
			fmt.Printf("Trained %s => %s\n", pattern, word)
		} else {
			logVarnamError()
		}
	} else if *learnFlag {
		word := args[0]

		if varnam.Learn(word, 0) {
			fmt.Printf("Learnt %s\n", word)
		} else {
			fmt.Printf("Couldn't learn %s", word)
			logVarnamError()
		}
	} else if *unlearnFlag {
		word := args[0]

		if varnam.Unlearn(word) {
			fmt.Printf("Unlearnt %s\n", word)
		} else {
			fmt.Printf("Couldn't learn %s", word)
			logVarnamError()
		}
	} else if *learnFromFileFlag {
		if varnam.LearnFromFile(args[0]) {
			fmt.Println("Finished learning from file")
		} else {
			logVarnamError()
		}
	} else if *trainFromFileFlag {
		if varnam.TrainFromFile(args[0]) {
			fmt.Println("Finished training from file")
		} else {
			logVarnamError()
		}
	} else if *exportFlag {
		if varnam.Export(args[0]) {
			fmt.Println("Finished exporting to file")
		} else {
			logVarnamError()
		}
	} else if *importFlag {
		if varnam.Import(args[0]) {
			fmt.Println("Finished importing from file")
		} else {
			logVarnamError()
		}
	} else if *reverseTransliterate {
		sugs, err := varnam.ReverseTransliterate(args[0])
		if err != nil {
			log.Fatal(err)
		}

		probMsg := false
		lastWeight := sugs[0].Weight

		fmt.Println("Exact Matches")
		for _, sug := range sugs {
			// The first exact matches will have same weight value
			if !probMsg && lastWeight != sug.Weight {
				fmt.Println("Probability Match")
				probMsg = true
			}
			fmt.Println(sug.Word + " " + fmt.Sprint(sug.Weight))
			lastWeight = sug.Weight
		}
	} else {
		var result govarnamgo.TransliterationResult

		if *greedy {
			// results = C.TransliterateGreedy(args[0])
		} else {
			result = varnam.Transliterate(context.Background(), args[0])
		}

		printSugs := func(sugs []govarnamgo.Suggestion) {
			for _, sug := range sugs {
				if sug.LearnedOn == 0 {
					fmt.Println(sug.Word + " " + fmt.Sprint(sug.Weight))
				} else {
					fmt.Println(sug.Word + " " + fmt.Sprint(sug.Weight) + " " + time.Unix(int64(sug.LearnedOn), 0).String())
				}
			}
		}

		fmt.Println("Greedy Tokenized")
		printSugs(result.GreedyTokenized)

		fmt.Println("Exact Matches")
		printSugs(result.ExactMatches)

		fmt.Println("Dictionary Suggestions")
		printSugs(result.DictionarySuggestions)

		fmt.Println("Pattern Dictionary Suggestions")
		printSugs(result.PatternDictionarySuggestions)

		fmt.Println("Tokenizer Suggestions")
		printSugs(result.TokenizerSuggestions)
	}
}
