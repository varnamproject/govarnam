package main

import (
	sql "database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

var (
	vstConn *sql.DB
)

// Token info for making a suggestion
type Token struct {
	tokenType int
	token     []Symbol
}

// Symbol result from VST
type Symbol struct {
	id              int
	generalType     int
	matchType       int
	pattern         string
	value1          string
	value2          string
	value3          string
	tag             string
	priority        int
	acceptCondition int
	flags           int
}

// Suggestion suggsestion
type Suggestion struct {
	word   string
	weight int
}

func openVST() {
	var err error
	vstConn, err = sql.Open("sqlite3", "./ml.vst")
	if err != nil {
		log.Fatal(err)
	}
}

func search(ch string) []Symbol {
	rows, err := vstConn.Query("select id, type, match_type, pattern, value1, value2, value3, tag, priority, accept_condition, flags from symbols where pattern = ? and match_type = 1;", ch)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	var results []Symbol

	for rows.Next() {
		var item Symbol
		rows.Scan(&item.id, &item.generalType, &item.matchType, &item.pattern, &item.value1, &item.value2, &item.value3, &item.tag, &item.priority, &item.acceptCondition, &item.flags)
		results = append(results, item)
	}

	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}

	return results
}

func make(word string) []Token {
	var results []Token

	var prevSequenceMatches []Symbol
	var sequence string

	i := 0
	for i < len(word) {
		ch := string(word[i])

		sequence += ch

		matches := search(sequence)
		fmt.Println(sequence, matches)

		if len(matches) == 0 {
			// No more matches

			if len(sequence) == 1 {
				// No matches for a single char, add it
				token := Token{VARNAM_TOKEN_CHAR, matches}
				results = append(results, token)
			} else {
				// Backtrack and add the previous sequence matches
				token := Token{VARNAM_TOKEN_SYMBOL, prevSequenceMatches}
				results = append(results, token)
				i--
			}

			sequence = ""
		} else {
			if i == len(word)-1 {
				// Last character
				token := Token{VARNAM_TOKEN_SYMBOL, matches}
				results = append(results, token)
			} else {
				prevSequenceMatches = matches
			}
		}
		i++
	}
	return results
}

func flatten(tokens []Token) []Suggestion {
	var results []Suggestion

	for i, t := range tokens {
		if t.tokenType == VARNAM_TOKEN_SYMBOL {
			if i == 0 {
				for _, possibility := range t.token {
					sug := Suggestion{possibility.value1, possibility.priority}
					results = append(results, sug)
				}
			} else {
				for j, result := range results {
					till := result.word
					tillWeight := result.weight

					results[j].word += t.token[0].value1
					results[j].weight += t.token[0].priority

					for k, possibility := range t.token {
						if k == 0 {
							continue
						}

						till += possibility.value1
						tillWeight += possibility.priority

						sug := Suggestion{till, tillWeight}
						results = append(results, sug)
					}
				}
			}
		}
	}

	return results
}

func transliterate(word string) {
	fmt.Println(flatten(make(word)))
}

func main() {
	openVST()
	transliterate(os.Args[1])
	defer vstConn.Close()
}
