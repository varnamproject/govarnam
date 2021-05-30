package main

import (
	"flag"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
	"gitlab.com/subins2000/govarnam/govarnam"
)

func main() {
	varnam := govarnam.Init("ml.vst", "./ml.vst.learnings")

	debugFlag := flag.Bool("debug", false, "Enable debugging outputs")
	learnFlag := flag.Bool("learn", false, "Learn a word")
	trainFlag := flag.Bool("train", false, "Train a word with a particular pattern. 2 Arguments: Pattern & Word")
	flag.Parse()

	varnam.Debug(*debugFlag)
	args := flag.Args()

	if *trainFlag {
		pattern := args[0]
		word := args[1]

		varnam.Train(pattern, word)

		fmt.Printf("Trained %s => %s", pattern, word)
	} else if *learnFlag {
		word := args[0]

		varnam.Learn(word)

		fmt.Printf("Learnt %s", word)
	} else {
		fmt.Println(varnam.Transliterate(args[0], 2))
	}

	defer varnam.Close()
}
