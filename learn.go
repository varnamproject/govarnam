package main

import (
	"context"
	"fmt"
	"log"
	"time"
)

type WordInfo struct {
	id         int
	word       string
	confidence int
	learnedOn  int
}

func checkError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

// Insert a word into word DB. Increment confidence if word exists
func insertWord(word string, confidence int) {
	query := "INSERT OR IGNORE INTO words(word, confidence, learned_on) VALUES (trim(?), ?, strftime('%s', datetime(), 'localtime'))"
	ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelfunc()
	stmt, err := dictConn.PrepareContext(ctx, query)
	checkError(err)
	defer stmt.Close()
	_, err = stmt.ExecContext(ctx, word, confidence)
	checkError(err)

	query = "UPDATE words SET confidence = confidence + 1 WHERE word = ?"
	ctx, cancelfunc = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelfunc()
	stmt, err = dictConn.PrepareContext(ctx, query)
	checkError(err)
	defer stmt.Close()
	_, err = stmt.ExecContext(ctx, word)
	checkError(err)
}

// Learns a word. If already exist, increases confidence of the pathway to that word.
// When learning a word, each path to that word is inserted into DB.
// Eg: ചങ്ങാതി: ചങ്ങ -> ചങ്ങാ -> ചങ്ങാതി
func learn(word string) {
	conjuncts := splitWordByConjunct(word)
	sequence := conjuncts[0]
	for i, ch := range conjuncts {
		if i == 0 {
			continue
		}
		sequence += ch
		if debug {
			fmt.Println(sequence)
		}
		// The final word should have the highest confidence
		insertWord(sequence, VARNAM_LEARNT_WORD_MIN_CONFIDENCE-(len(conjuncts)-1-i))
	}
}

// Unlearn a word, remove from words DB and pattern if there is
func unlearn(word string) {
	// TODO
}

// Train a word with a particular pattern. Pattern => word
func train(pattern string, word string) {
	learn(word)

	wordInfo := getWordInfo(word)

	query := "INSERT OR IGNORE INTO patterns_content(pattern, word_id, learned) VALUES (?, ?, 1)"
	ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelfunc()
	stmt, err := dictConn.PrepareContext(ctx, query)
	checkError(err)
	defer stmt.Close()
	_, err = stmt.ExecContext(ctx, pattern, wordInfo.id)
	checkError(err)
}

func getWordInfo(word string) *WordInfo {
	rows, err := dictConn.Query("SELECT id, confidence, learned_on FROM words WHERE word = ?", word)
	checkError(err)
	defer rows.Close()

	var wordInfo WordInfo
	wordExists := false

	for rows.Next() {
		// This loop will only work if there is a word
		wordExists = true

		rows.Scan(&wordInfo.id, &wordInfo.confidence, &wordInfo.learnedOn)
	}

	if wordExists {
		return &wordInfo
	} else {
		return nil
	}
}
