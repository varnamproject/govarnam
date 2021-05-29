package main

import (
	"context"
	"log"
	"time"
)

type WordInfo struct {
	id         int
	word       string
	confidence int
}

func checkError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func train(pattern string, word string) {
	var wordInfo = getWordInfo(word)

	if wordInfo == nil {
		query := "INSERT INTO words(word, confidence) VALUES (?, 1)"
		ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelfunc()
		stmt, err := dictConn.PrepareContext(ctx, query)
		checkError(err)
		defer stmt.Close()
		_, err = stmt.ExecContext(ctx, word)
		checkError(err)

		wordInfo = getWordInfo(word)
	}

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
	rows, err := dictConn.Query("SELECT id, confidence FROM words WHERE word = ?", word)
	checkError(err)
	defer rows.Close()

	var wordInfo WordInfo
	wordExists := false

	for rows.Next() {
		// This loop will only work if there is a word
		wordExists = true

		rows.Scan(&wordInfo.id, &wordInfo.confidence)
	}

	if wordExists {
		return &wordInfo
	} else {
		return nil
	}
}
