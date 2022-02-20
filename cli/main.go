package main

/**
 * govarnam - An Indian language transliteration library
 * Copyright Subin Siby <mail at subinsb (.) com>, 2021
 * Licensed under AGPL-3.0-only. See LICENSE.txt
 */

import (
	"context"
	"flag"
	"fmt"
	"log"
	"path/filepath"
	"time"

	"github.com/varnamproject/govarnam/govarnamgo"
)

var varnam *govarnamgo.VarnamHandle

func printSugs(sugs []govarnamgo.Suggestion) {
	for _, sug := range sugs {
		if sug.LearnedOn == 0 {
			fmt.Println(sug.Word + " " + fmt.Sprint(sug.Weight))
		} else {
			fmt.Println(sug.Word + " " + fmt.Sprint(sug.Weight) + " " + time.Unix(int64(sug.LearnedOn), 0).String())
		}
	}
}

func main() {
	versionFlag := flag.Bool("version", false, "Show version information")

	debugFlag := flag.Bool("debug", false, "Enable debugging outputs")
	schemeFlag := flag.String("s", "", "Scheme ID")

	reIndexFlag := flag.Bool("reindex", false, "Reindex user dictionary database")

	learnFlag := flag.Bool("learn", false, "Learn a word")
	unlearnFlag := flag.Bool("unlearn", false, "Unlearn a word")
	trainFlag := flag.Bool("train", false, "Train a word with a particular pattern. 2 Arguments: Pattern & Word")

	learnFromFileFlag := flag.Bool("learn-from-file", false, "Learn words in a file")
	trainFromFileFlag := flag.Bool("train-from-file", false, "Train pattern => word from a file.")

	exportFlag := flag.Bool("export", false, "Export learnings to file")
	exportWordsPerFile := flag.Int("export-words-per-file", 30000, "Words per export file")
	importFlag := flag.Bool("import", false, "Import learnings from file")

	indicDigitsFlag := flag.Bool("digits", false, "Use indic digits")

	advanced := flag.Bool("advanced", false, "Show transliteration result in advanced mode")
	reverseTransliterate := flag.Bool("reverse", false, "Reverse transliterate. Find which pattern to use for a specific word")

	flag.Parse()

	if *versionFlag {
		fmt.Println(govarnamgo.GetVersion())
		fmt.Println(govarnamgo.GetBuild())
		return
	}

	if *schemeFlag == "" {
		fmt.Println("Specifiy a scheme ID with -s.\n\nUse --help for all available commands.")
		return
	}

	var err error
	varnam, err = govarnamgo.InitFromID(*schemeFlag)
	if err != nil {
		log.Fatal(err.Error())
	}

	varnam.Debug(*debugFlag)

	config := govarnamgo.Config{IndicDigits: *indicDigitsFlag, DictionarySuggestionsLimit: 10, PatternDictionarySuggestionsLimit: 10, TokenizerSuggestionsLimit: 10, TokenizerSuggestionsAlways: true}
	varnam.SetConfig(config)

	args := flag.Args()

	if *reIndexFlag {
		err := varnam.ReIndexDictionary()
		if err != nil {
			log.Fatal(err.Error())
		}
		fmt.Println("Successfully re-indexed dictionary.")
	} else if *trainFlag {
		pattern := args[0]
		word := args[1]

		err := varnam.Train(pattern, word)
		if err != nil {
			log.Fatal(err.Error())
		}
		fmt.Printf("Trained %s => %s\n", pattern, word)
	} else if *learnFlag {
		word := args[0]

		err := varnam.Learn(word, 0)
		if err == nil {
			fmt.Printf("Learnt %s\n", word)
		} else {
			fmt.Printf("Couldn't learn %s", word)
			log.Fatal(err.Error())
		}
	} else if *unlearnFlag {
		word := args[0]

		err := varnam.Unlearn(word)
		if err == nil {
			fmt.Printf("Unlearnt %s\n", word)
		} else {
			fmt.Printf("Couldn't learn %s", word)
			log.Fatal(err.Error())
		}
	} else if *learnFromFileFlag {
		learnStatus, err := varnam.LearnFromFile(args[0])
		if err == nil {
			fmt.Printf("Finished learning from file. Total words: %d. Failed: %d\n", learnStatus.TotalWords, learnStatus.FailedWords)
		} else {
			log.Fatal(err.Error())
		}
	} else if *trainFromFileFlag {
		learnStatus, err := varnam.TrainFromFile(args[0])
		if err == nil {
			fmt.Printf("Finished training from file. Total words: %d. Failed: %d\n", learnStatus.TotalWords, learnStatus.FailedWords)
		} else {
			log.Fatal(err.Error())
		}
	} else if *exportFlag {
		err := varnam.Export(args[0], *exportWordsPerFile)
		if err == nil {
			fmt.Println("Finished exporting to file")
		} else {
			log.Fatal(err.Error())
		}
	} else if *importFlag {
		matches, err := filepath.Glob(args[0])

		if err != nil {
			log.Fatal(err.Error())
		}

		for _, match := range matches {
			err := varnam.Import(match)
			if err == nil {
				fmt.Printf("Finished importing from file %s\n", match)
			} else {
				log.Fatal(err.Error())
			}
		}
	} else if *reverseTransliterate {
		sugs, err := varnam.ReverseTransliterate(args[0])
		if err != nil {
			log.Fatal(err.Error())
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
	} else if *advanced {
		var result govarnamgo.TransliterationResult

		result, _ = varnam.TransliterateAdvanced(context.Background(), args[0])

		fmt.Println("Greedy Tokenized")
		printSugs(result.GreedyTokenized)

		fmt.Println("Exact Words")
		printSugs(result.ExactWords)

		fmt.Println("Exact Matches")
		printSugs(result.ExactMatches)

		fmt.Println("Dictionary Suggestions")
		printSugs(result.DictionarySuggestions)

		fmt.Println("Pattern Dictionary Suggestions")
		printSugs(result.PatternDictionarySuggestions)

		fmt.Println("Tokenizer Suggestions")
		printSugs(result.TokenizerSuggestions)
	} else {
		result, _ := varnam.Transliterate(context.Background(), args[0])
		printSugs(result)
	}

	varnam.Close()
}
