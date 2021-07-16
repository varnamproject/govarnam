package govarnam

import (
	"context"
	"log"
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
	varnam.dictConn, err = openDB(dictPath)
	return err
}

func makeDictionary(dictPath string) error {
	conn, err := openDB(dictPath)
	if err != nil {
		return err
	}
	defer conn.Close()

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
			return err
		}
		defer stmt.Close()

		_, err = stmt.ExecContext(ctx)
		if err != nil {
			return err
		}
	}

	return nil
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
			likes += "OR word LIKE ? "
			if all == true {
				vals = append(vals, word+"_%")
			} else {
				vals = append(vals, word)
			}
		}

		if all == true {
			query = "SELECT word, confidence, learned_on FROM words WHERE word LIKE ? " + likes + " AND learned_on > 0 ORDER BY confidence DESC LIMIT ?"
		} else {
			query = "SELECT word, confidence, learned_on FROM words WHERE word LIKE ? " + likes + " ORDER BY confidence DESC LIMIT 5"
		}
		vals = append(vals, varnam.DictionarySuggestionsLimit)

		rows, err := varnam.dictConn.QueryContext(ctx, query, vals...)

		if err != nil {
			log.Print(err)
			return results
		}

		defer rows.Close()

		for rows.Next() {
			var item Suggestion
			rows.Scan(&item.Word, &item.Weight, &item.LearnedOn)
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
					for _, possibility := range t.symbols {
						// Weight has no use in dictionary lookup
						sug := Suggestion{getSymbolValue(possibility, 0), 0, 0}

						search := []string{sug.Word}
						searchResults := varnam.searchDictionary(ctx, search, false)

						if len(searchResults) > 0 {
							tempFoundDictWords = append(tempFoundDictWords, searchResults[0])
							results = append(results, sug)
						}
					}
				} else {
					for j, result := range results {
						if result.Weight == -1 {
							continue
						}

						till := result.Word

						firstSymbol := t.symbols[0]
						results[j].Word += getSymbolValue(firstSymbol, i)

						search := []string{results[j].Word}
						searchResults := varnam.searchDictionary(ctx, search, false)

						if len(searchResults) > 0 {
							tempFoundDictWords = append(tempFoundDictWords, searchResults[0])
						} else {
							// No need of processing this anymore.
							// Weight is used as a flag here to skip some results
							results[j].Weight = -1
						}

						for k, symbol := range t.symbols {
							if k == 0 {
								continue
							}

							newTill := till + getSymbolValue(symbol, i)

							search = []string{newTill}
							searchResults = varnam.searchDictionary(ctx, search, false)

							if len(searchResults) > 0 {
								tempFoundDictWords = append(tempFoundDictWords, searchResults[0])

								sug := Suggestion{newTill, 0, 0}
								results = append(results, sug)
							}
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
