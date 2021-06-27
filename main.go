package main

import (
	"flag"
	"fmt"
	"log"

	"gitlab.com/subins2000/govarnam/govarnam"
)

func main() {
	debugFlag := flag.Bool("debug", false, "Enable debugging outputs")
	langFlag := flag.String("lang", "", "Language")

	learnFlag := flag.Bool("learn", false, "Learn a word")
	unlearnFlag := flag.Bool("unlearn", false, "Unlearn a word")
	trainFlag := flag.Bool("train", false, "Train a word with a particular pattern. 2 Arguments: Pattern & Word")

	indicDigitsFlag := flag.Bool("digits", false, "Use indic digits")
	greedy := flag.Bool("greedy", false, "Show only exactly matched suggestions")

	flag.Parse()

	varnam, err := govarnam.InitFromLang(*langFlag)

	if err != nil {
		log.Fatal(err)
		return
	}

	varnam.Debug = *debugFlag
	varnam.LangRules.IndicDigits = *indicDigitsFlag

	args := flag.Args()

	if *trainFlag {
		pattern := args[0]
		word := args[1]

		err := varnam.Train(pattern, word)
		if err == nil {
			fmt.Printf("Trained %s => %s\n", pattern, word)
		} else {
			fmt.Printf(err.Error() + "\n")
		}
	} else if *learnFlag {
		word := args[0]

		varnam.Learn(word)

		fmt.Printf("Learnt %s", word)
	} else if *unlearnFlag {
		word := args[0]

		varnam.Unlearn(word)

		fmt.Printf("Unlearnt %s", word)
	} else {
		var results govarnam.TransliterationResult

		if *greedy {
			results = varnam.TransliterateGreedy(args[0])
		} else {
			results = varnam.Transliterate(args[0])
		}

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

	defer varnam.Close()
}
