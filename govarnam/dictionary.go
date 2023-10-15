package govarnam

/**
 * govarnam - An Indian language transliteration library
 * Copyright Subin Siby <mail at subinsb (.) com>, 2021
 * Licensed under AGPL-3.0-only. See LICENSE.txt
 */

import (
	"context"
	sql "database/sql"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path"
	"time"
)

//go:embed migrations/*.sql
var embedFS embed.FS

// DictionaryResult result from dictionary search
type DictionaryResult struct {
	// Exactly found starting word matches.
	exactMatches []Suggestion

	// Words found in dictionary with same starting
	partialMatches []Suggestion

	longestMatchPosition int
}

// MoreDictionaryResult result from dictionary search
type MoreDictionaryResult struct {
	// Exactly found words
	exactWords []Suggestion

	// Words found in dictionary with same starting
	moreSuggestions [][]Suggestion
}

// PatternDictionarySuggestion longest match result
type PatternDictionarySuggestion struct {
	Sug    Suggestion
	Length int
}

type searchDictionaryResult struct {
	match     string
	word      string
	weight    int
	learnedOn int
}

// InitDict open connection to dictionary
func (varnam *Varnam) InitDict(dictConfig *DictionaryConfig) error {
	var err error

	if !fileExists(dictConfig.Path) {
		log.Printf("Making Varnam dictionaries dir for %s\n", dictConfig.Path)
		err := os.MkdirAll(path.Dir(dictConfig.Path), 0750)
		if err != nil {
			return err
		}
	}

	dictConfig.conn, err = openDB(dictConfig.Path)
	if err != nil {
		return err
	}

	if !dictConfig.Write {
		return nil
	}

	// cd into migrations directory
	migrationsFS, err := fs.Sub(embedFS, "migrations")
	if err != nil {
		return err
	}

	mg, err := InitMigrate(dictConfig.conn, migrationsFS)
	if err != nil {
		return err
	}

	ranMigrations, err := mg.Run()
	if ranMigrations != 0 {
		log.Printf("ran %d migrations", ranMigrations)
	}

	// Since SQLite v3.12.0, default page size is 4096
	dictConfig.conn.Exec("PRAGMA page_size=4096;")
	// WAL makes writes & reads happen concurrently => significantly fast
	dictConfig.conn.Exec("PRAGMA journal_mode=wal;")

	return err
}

// ReIndexDictionary re-indexes dictionary
func (varnam *Varnam) ReIndexDictionary() error {
	for _, dictConfig := range varnam.DictsConfig {
		if !dictConfig.Write {
			continue
		}

		_, err := dictConfig.conn.Exec("INSERT INTO words_fts(words_fts) VALUES('rebuild');")
		if err != nil {
			return err
		}
	}
	return nil
}

type searchDictionaryType int32

const (
	searchMatches      searchDictionaryType = 0 // For checking whether there are words in dictionary starting with something
	searchStartingWith searchDictionaryType = 1 // Find all words in dictionary starting with something
	searchExactWords   searchDictionaryType = 2 // Find exact words in dictionary
)

// all - Search for words starting with the word
func (varnam *Varnam) searchDictionary(
	ctx context.Context,
	dbConn *sql.DB,
	words []string,
	searchType searchDictionaryType,
) []searchDictionaryResult {
	likes := ""

	var (
		vals    []interface{}
		query   string
		results []searchDictionaryResult
	)

	select {
	case <-ctx.Done():
		return results
	default:
		if searchType == searchExactWords {
			vals = append(vals, words[0])
		} else {
			// FTS5 MATCH requires strings to be wrapped in double quotes
			// https://stackoverflow.com/q/28971633
			// https://github.com/varnamproject/govarnam/issues/27
			vals = append(vals, "\""+words[0]+"\"")
		}

		for i := range words {
			if i == 0 {
				continue
			}
			likes += ", (?)"

			if searchType == searchExactWords {
				vals = append(vals, words[i])
			} else {
				vals = append(vals, "\""+words[i]+"\"")
			}
		}

		// Thanks forpas
		// CC BY-SA 4.0 licensed
		// https://stackoverflow.com/q/68610241/1372424

		if searchType == searchMatches {
			query = `
				WITH cte(match) AS (VALUES (?) ` + likes + `)
				SELECT
					SUBSTR(c.match, 2, LENGTH(c.match) - 2) AS match, -- Result will be double quoted, remove it
					w.word AS word,
					MAX(w.weight),
					MAX(w.learned_on)
				FROM words_fts w
				INNER JOIN cte c
					ON w.word MATCH c.match || '*'
				GROUP BY c.match
				`
		} else if searchType == searchStartingWith {
			query = `
				WITH cte(match) AS (VALUES (?) ` + likes + `)
				SELECT
					SUBSTR(c.match, 2, LENGTH(c.match) - 1) AS match,
					w.*
				FROM words_fts w
				INNER JOIN cte c
					ON w.word MATCH c.match || '*'
					AND w.word != c.match
				ORDER BY weight DESC LIMIT ?
				`
			vals = append(vals, varnam.DictionarySuggestionsLimit)
		} else if searchType == searchExactWords {
			query = "SELECT * FROM words WHERE word IN ((?) " + likes + ")"
		}

		rows, err := dbConn.QueryContext(ctx, query, vals...)

		if err != nil {
			log.Print(err)
			return results
		}

		defer rows.Close()

		for rows.Next() {
			var item searchDictionaryResult
			rows.Scan(&item.match, &item.word, &item.weight, &item.learnedOn)
			results = append(results, item)
		}

		err = rows.Err()
		if err != nil {
			log.Print(err)
			return results
		}

		return results
	}
}

func (varnam *Varnam) getFromDictionary(
	ctx context.Context,
	dbConn *sql.DB,
	tokensPointer *[]Token,
) DictionaryResult {
	var result DictionaryResult
	tokens := *tokensPointer

	select {
	case <-ctx.Done():
		return result
	default:
		// This is a temporary storage for words made from tokens,
		// which will be searched in dictionary.
		// Similar to 'result' usage in tokenizeWord
		var tokenizedWords []searchDictionaryResult

		// We search in dictionary by going through each token,
		// these vars would store the last found results
		var lastFoundDictWords []searchDictionaryResult
		var lastFoundPosition = 0

		for i, t := range tokens {
			var tempFoundDictWords []searchDictionaryResult
			if t.tokenType == VARNAM_TOKEN_SYMBOL {
				if i == 0 {
					start := time.Now()

					var toSearch []string
					for j := range t.symbols {
						toSearch = append(toSearch, getSymbolValue(t.symbols[j], 0))
					}

					searchResults := varnam.searchDictionary(
						ctx,
						dbConn,
						toSearch,
						searchMatches,
					)

					tempFoundDictWords = searchResults
					tokenizedWords = searchResults

					if LOG_TIME_TAKEN {
						log.Printf(
							"%s took %v\n",
							"getFromDictionaryToken0",
							time.Since(start),
						)
					}
				} else {
					start := time.Now()
					for j := range tokenizedWords {
						if tokenizedWords[j].weight == -1 {
							continue
						}

						till := tokenizedWords[j].match

						var toSearch []string

						for _, symbol := range t.symbols {
							newTill := till + getSymbolValue(symbol, i)
							toSearch = append(toSearch, newTill)
						}

						searchResults := varnam.searchDictionary(
							ctx,
							dbConn,
							toSearch,
							searchMatches,
						)

						if len(searchResults) > 0 {
							tempFoundDictWords = append(tempFoundDictWords, searchResults...)

							for k := range searchResults {
								if k == 0 {
									tokenizedWords[j].match = searchResults[k].match
									continue
								}

								sug := searchDictionaryResult{
									searchResults[k].match,
									searchResults[k].word,
									0,
									0,
								}
								tokenizedWords = append(tokenizedWords, sug)
							}
						} else {
							// No need of processing this word anymore, we found no match in dictionary.
							// weight is used as a flag here to skip processing this further.
							tokenizedWords[j].weight = -1
						}
					}
					if LOG_TIME_TAKEN {
						log.Printf("%s%d took %v\n", "getFromDictionaryToken", i, time.Since(start))
					}
				}
			}
			if len(tempFoundDictWords) > 0 {
				lastFoundDictWords = tempFoundDictWords
				lastFoundPosition = t.position
			}
		}

		if lastFoundPosition == tokens[len(tokens)-1].position {
			result.exactMatches = convertSearchDictResultToSuggestions(lastFoundDictWords, false)
		} else {
			result.partialMatches = convertSearchDictResultToSuggestions(lastFoundDictWords, false)
		}

		result.longestMatchPosition = lastFoundPosition

		return result
	}
}

func (varnam *Varnam) getMoreFromDictionary(
	ctx context.Context,
	dbConn *sql.DB,
	words []Suggestion,
) MoreDictionaryResult {
	var result MoreDictionaryResult

	select {
	case <-ctx.Done():
		return result
	default:
		wordsToSearch := []string{}

		for i := range words {
			wordsToSearch = append(wordsToSearch, words[i].Word)

			search := []string{words[i].Word}
			result.moreSuggestions = append(
				result.moreSuggestions,
				convertSearchDictResultToSuggestions(
					varnam.searchDictionary(ctx, dbConn, search, searchStartingWith),
					true,
				),
			)
		}

		result.exactWords = convertSearchDictResultToSuggestions(
			varnam.searchDictionary(ctx, dbConn, wordsToSearch, searchExactWords),
			true,
		)

		return result
	}
}

// Gets incomplete and complete matches from pattern dictionary
// Eg: If pattern = "chin" or "chinayil", will return "china"
func (varnam *Varnam) getFromPatternDictionary(
	ctx context.Context,
	dbConn *sql.DB,
	pattern string,
) []PatternDictionarySuggestion {
	var results []PatternDictionarySuggestion

	select {
	case <-ctx.Done():
		return results
	default:
		rows, err := dbConn.QueryContext(ctx, "SELECT LENGTH(pts.pattern), w.word, w.weight, w.learned_on FROM `patterns` pts LEFT JOIN words w ON w.id = pts.word_id WHERE ? LIKE (pts.pattern || '%') OR pattern LIKE ? ORDER BY LENGTH(pts.pattern) DESC LIMIT ?", pattern, pattern+"%", varnam.PatternDictionarySuggestionsLimit)

		if err != nil {
			log.Print(err)
			return results
		}

		defer rows.Close()

		for rows.Next() {
			var item PatternDictionarySuggestion
			rows.Scan(&item.Length, &item.Sug.Word, &item.Sug.Weight, &item.Sug.LearnedOn)
			item.Sug.Weight += VARNAM_LEARNT_WORD_MIN_WEIGHT
			results = append(results, item)
		}

		err = rows.Err()
		if err != nil {
			log.Print(err)
		}

		return results
	}
}

// get recently learnt words
func (varnam *Varnam) getRecentlyLearntWordsFromDict(
	ctx context.Context,
	dbConn *sql.DB,
	offset int,
	limit int,
) ([]Suggestion, error) {
	var result []Suggestion

	select {
	case <-ctx.Done():
		return result, nil
	default:
		rows, err := dbConn.QueryContext(ctx, "SELECT word, weight, learned_on FROM words ORDER BY learned_on DESC, id DESC LIMIT "+fmt.Sprint(offset)+", "+fmt.Sprint(limit))

		if err != nil {
			return result, err
		}
		defer rows.Close()

		for rows.Next() {
			var item Suggestion
			rows.Scan(&item.Word, &item.Weight, &item.LearnedOn)
			result = append(result, item)
		}

		err = rows.Err()
		if err != nil {
			log.Print(err)
			return result, err
		}

		return result, nil
	}
}

// get word suggestions from dictionary
func (varnam *Varnam) getSuggestionsFromDict(ctx context.Context, dbConn *sql.DB, word string) []Suggestion {
	var sugs []Suggestion

	select {
	case <-ctx.Done():
		return sugs
	default:
		return convertSearchDictResultToSuggestions(
			varnam.searchDictionary(ctx, dbConn, []string{word}, searchStartingWith),
			true,
		)
	}
}

func convertSearchDictResultToSuggestions(searchResults []searchDictionaryResult, word bool) []Suggestion {
	var sugs []Suggestion
	for i := range searchResults {
		sug := Suggestion{
			searchResults[i].match,
			searchResults[i].weight,
			searchResults[i].learnedOn,
		}
		if word {
			sug.Word = searchResults[i].word
		}
		sugs = append(sugs, sug)
	}
	return sugs
}
