package govarnam

import (
	"context"
	sql "database/sql"
	"log"
	"os"
	"path"
	"time"
)

// DictionaryResult result from dictionary search
type DictionaryResult struct {
	sugs                 []Suggestion
	exactMatch           bool
	longestMatchPosition int
}

// PatternDictionarySuggestion longest match result
type PatternDictionarySuggestion struct {
	Sug    Suggestion
	Length int
}

// InitDict open connection to dictionary
func (varnam *Varnam) InitDict(dictPath string) error {
	var err error

	if !fileExists(dictPath) {
		log.Printf("Making Varnam Learnings File at %s\n", dictPath)
		os.MkdirAll(path.Dir(dictPath), 0750)

		varnam.dictConn, err = makeDictionary(dictPath)
	} else {
		varnam.dictConn, err = openDB(dictPath)
	}

	if err == nil {
		varnam.dictConn.Exec("PRAGMA TEMP_STORE=2;")
		varnam.dictConn.Exec("PRAGMA LOCKING_MODE=EXCLUSIVE;")
	}

	return err
}

func makeDictionary(dictPath string) (*sql.DB, error) {
	conn, err := openDB(dictPath)
	if err != nil {
		return nil, err
	}

	conn.Exec("PRAGMA page_size=4096;")
	conn.Exec("PRAGMA journal_mode=wal;")

	queries := [3]string{"CREATE TABLE IF NOT EXISTS metadata (key TEXT UNIQUE, value TEXT);",
		"CREATE TABLE IF NOT EXISTS words (id integer primary key, word text unique, confidence integer default 1, learned_on integer);",
		"CREATE TABLE IF NOT EXISTS patterns_content ( `pattern` text, `word_id` integer, FOREIGN KEY(`word_id`) REFERENCES `words`(`id`) ON DELETE CASCADE, PRIMARY KEY(`pattern`,`word_id`) ) WITHOUT ROWID;"}

	for _, query := range queries {
		ctx, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelFunc()

		stmt, err := conn.PrepareContext(ctx, query)
		if err != nil {
			return nil, err
		}
		defer stmt.Close()

		_, err = stmt.ExecContext(ctx)
		if err != nil {
			return nil, err
		}
	}

	return conn, nil
}

// all - Search for words starting with the word
func (varnam *Varnam) searchDictionary(ctx context.Context, words []string, all bool) []Suggestion {
	likes := ""

	var (
		vals    []interface{}
		query   string
		results []Suggestion
	)

	select {
	case <-ctx.Done():
		return results
	default:
		if all == true {
			// _% means a wildcard with a sequence of 1 or more
			// % means 0 or more and would include the word itself
			vals = append(vals, words[0]+"_%")
		} else {
			vals = append(vals, words[0])
		}

		for i, word := range words {
			if i == 0 {
				continue
			}
			if all == true {
				likes += "OR word LIKE ? "
				vals = append(vals, word+"_%")
			} else {
				likes += ", (?)"
				vals = append(vals, word)
			}
		}

		if all == true {
			query = "SELECT word, confidence, learned_on FROM words WHERE word LIKE ? " + likes + " AND learned_on > 0 ORDER BY confidence DESC LIMIT ?"
			vals = append(vals, varnam.DictionarySuggestionsLimit)
		} else {
			query = "WITH cte(match) AS (VALUES (?) " + likes + ") SELECT DISTINCT c.match AS word FROM words w INNER JOIN cte c ON w.word LIKE c.match || '%'"
		}

		rows, err := varnam.dictConn.QueryContext(ctx, query, vals...)

		if err != nil {
			log.Print(err)
			return results
		}

		defer rows.Close()

		for rows.Next() {
			var item Suggestion
			if all {
				rows.Scan(&item.Word, &item.Weight, &item.LearnedOn)
			} else {
				rows.Scan(&item.Word)
			}
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

func (varnam *Varnam) getFromDictionary(ctx context.Context, tokensPointer *[]Token) DictionaryResult {
	var endResult DictionaryResult
	tokens := *tokensPointer

	select {
	case <-ctx.Done():
		return endResult
	default:
		// This is a temporary storage for tokenized words
		// Similar to usage in tokenizeWord
		var results []Suggestion

		foundPosition := 0
		var foundDictWords []Suggestion

		for i, t := range tokens {
			var tempFoundDictWords []Suggestion
			if t.tokenType == VARNAM_TOKEN_SYMBOL {
				if i == 0 {
					var toSearch []string
					for _, possibility := range t.symbols {
						toSearch = append(toSearch, getSymbolValue(possibility, 0))
					}

					searchResults := varnam.searchDictionary(ctx, toSearch, false)

					tempFoundDictWords = searchResults
					results = searchResults
				} else {
					for j, result := range results {
						if result.Weight == -1 {
							continue
						}

						till := result.Word

						var toSearch []string

						for _, symbol := range t.symbols {
							newTill := till + getSymbolValue(symbol, i)
							toSearch = append(toSearch, newTill)
						}

						searchResults := varnam.searchDictionary(ctx, toSearch, false)

						if len(searchResults) > 0 {
							tempFoundDictWords = append(tempFoundDictWords, searchResults...)

							for k, searchResult := range searchResults {
								if k == 0 {
									results[j].Word = searchResult.Word
									continue
								}

								sug := Suggestion{searchResult.Word, 0, 0}
								results = append(results, sug)
							}
						} else {
							// No need of processing this anymore.
							// Weight is used as a flag here to skip some results
							results[j].Weight = -1
						}
					}
				}
			}
			if len(tempFoundDictWords) > 0 {
				foundDictWords = tempFoundDictWords
				foundPosition = t.position
			}
		}

		endResult.sugs = foundDictWords
		endResult.exactMatch = foundPosition == tokens[len(tokens)-1].position
		endResult.longestMatchPosition = foundPosition

		return endResult
	}
}

func (varnam *Varnam) getMoreFromDictionary(ctx context.Context, words []Suggestion) [][]Suggestion {
	var results [][]Suggestion

	select {
	case <-ctx.Done():
		return results
	default:
		for _, sug := range words {
			search := []string{sug.Word}
			searchResults := varnam.searchDictionary(ctx, search, true)
			results = append(results, searchResults)
		}
		return results
	}
}

// A simpler function to get matches from pattern dictionary
// Gets incomplete matches.
// Eg: If pattern = "chin", will return "china"
// TODO better function name ? Ambiguous ?
func (varnam *Varnam) getTrailingFromPatternDictionary(ctx context.Context, pattern string) []Suggestion {
	var results []Suggestion

	select {
	case <-ctx.Done():
		return results
	default:
		rows, err := varnam.dictConn.QueryContext(ctx, "SELECT word, confidence FROM words WHERE id IN (SELECT word_id FROM patterns_content WHERE pattern LIKE ?) ORDER BY confidence DESC LIMIT 10", pattern+"%")

		if err != nil {
			log.Print(err)
			return results
		}

		defer rows.Close()

		for rows.Next() {
			var item Suggestion
			rows.Scan(&item.Word, &item.Weight)
			item.Weight += VARNAM_LEARNT_WORD_MIN_CONFIDENCE
			results = append(results, item)
		}

		err = rows.Err()
		if err != nil {
			log.Print(err)
		}

		return results
	}
}

// Gets incomplete and complete matches from pattern dictionary
// Eg: If pattern = "chin" or "chinayil", will return "china"
func (varnam *Varnam) getFromPatternDictionary(ctx context.Context, pattern string) []PatternDictionarySuggestion {
	var results []PatternDictionarySuggestion

	select {
	case <-ctx.Done():
		return results
	default:
		rows, err := varnam.dictConn.QueryContext(ctx, "SELECT LENGTH(pts.pattern), words.word, words.confidence, words.learned_on FROM `patterns_content` pts LEFT JOIN words ON words.id = pts.word_id WHERE ? LIKE (pts.pattern || '%') OR pattern LIKE ? ORDER BY LENGTH(pts.pattern) DESC LIMIT ?", pattern, pattern+"%", varnam.PatternDictionarySuggestionsLimit)

		if err != nil {
			log.Print(err)
			return results
		}

		defer rows.Close()

		for rows.Next() {
			var item PatternDictionarySuggestion
			rows.Scan(&item.Length, &item.Sug.Word, &item.Sug.Weight, &item.Sug.LearnedOn)
			item.Sug.Weight += VARNAM_LEARNT_WORD_MIN_CONFIDENCE
			results = append(results, item)
		}

		err = rows.Err()
		if err != nil {
			log.Print(err)
		}

		return results
	}
}
