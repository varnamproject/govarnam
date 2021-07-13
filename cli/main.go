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

	"gitlab.com/subins2000/govarnam/govarnamgo"
)

var varnam *govarnamgo.VarnamHandle

func logVarnamError() {
	log.Fatal(varnam.GetLastError())
}

func main() {
	debugFlag := flag.Bool("debug", false, "Enable debugging outputs")
	langFlag := flag.String("lang", "", "Language")

	learnFlag := flag.Bool("learn", false, "Learn a word")
	unlearnFlag := flag.Bool("unlearn", false, "Unlearn a word")
	trainFlag := flag.Bool("train", false, "Train a word with a particular pattern. 2 Arguments: Pattern & Word")

	learnFromFileFlag := flag.Bool("learn-from-file", false, "Learn words in a file")
	trainFromFileFlag := flag.Bool("train-from-file", false, "Train pattern => word from a file.")

	exportFlag := flag.Bool("export", false, "Export learnings to file")
	importFlag := flag.Bool("import", false, "Import learnings from file")

	indicDigitsFlag := flag.Bool("digits", false, "Use indic digits")
	greedy := flag.Bool("greedy", false, "Show only exactly matched suggestions")

	flag.Parse()

	if *langFlag == "" {
		log.Fatal("Specifiy a language with -lang")
	}

	var err error
	varnam, err = govarnamgo.InitFromID(*langFlag)
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
	} else {
		var result govarnamgo.TransliterationResult

		if *greedy {
			// results = C.TransliterateGreedy(args[0])
		} else {
			result = varnam.Transliterate(context.Background(), args[0])
		}
		fmt.Println("Greedy Tokenized")
		for _, sug := range result.GreedyTokenized {
			fmt.Println(sug.Word + " " + fmt.Sprint(sug.Weight))
		}

		fmt.Println("Exact Matches")
		for _, sug := range result.ExactMatches {
			fmt.Println(sug.Word + " " + fmt.Sprint(sug.Weight))
		}

		fmt.Println("Dictionary Suggestions")
		for _, sug := range result.DictionarySuggestions {
			fmt.Println(sug.Word + " " + fmt.Sprint(sug.Weight))
		}

		fmt.Println("Pattern Dictionary Suggestions")
		for _, sug := range result.PatternDictionarySuggestions {
			fmt.Println(sug.Word + " " + fmt.Sprint(sug.Weight))
		}

		fmt.Println("Tokenizer Suggestions")
		for _, sug := range result.TokenizerSuggestions {
			fmt.Println(sug.Word + " " + fmt.Sprint(sug.Weight))
		}
	}
}
