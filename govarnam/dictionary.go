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

	return err
}

func makeDictionary(dictPath string) (*sql.DB, error) {
	conn, err := openDB(dictPath)
	if err != nil {
		return nil, err
	}

	conn.Exec("PRAGMA page_size=4096;")
	conn.Exec("PRAGMA journal_mode=wal;")

	queries := [5]string{
		`
		CREATE TABLE IF NOT EXISTS metadata (
			key TEXT UNIQUE,
			value TEXT
		);
		`,
		`
		CREATE TABLE IF NOT EXISTS words (
			id INTEGER PRIMARY KEY,
			word TEXT UNIQUE,
			weight INTEGER DEFAULT 1,
			learned_on INTEGER
		);
		`,
		`
		CREATE VIRTUAL TABLE IF NOT EXISTS words_fts USING FTS5(
			word,
			weight UNINDEXED,
			learned_on UNINDEXED,
			content='words',
			content_rowid='id',
			tokenize='ascii',
			prefix='1 2',
		);
		`,
		`
		CREATE TRIGGER words_ai AFTER INSERT ON words
			BEGIN
				INSERT INTO words_fts (rowid, word)
				VALUES (new.id, new.word);
			END;
		
		CREATE TRIGGER words_ad AFTER DELETE ON words
			BEGIN
				INSERT INTO words_fts (words_fts, rowid, word)
				VALUES ('delete', old.id, old.word);
			END;
		
		CREATE TRIGGER words_au AFTER UPDATE ON words
			BEGIN
				INSERT INTO words_fts (words_fts, rowid, word)
				VALUES ('delete', old.id, old.word);
				INSERT INTO words_fts (rowid, word)
				VALUES (new.id, new.word);
			END;
		`,
		`
		CREATE TABLE IF NOT EXISTS patterns (
			pattern TEXT,
			word_id INTEGER,
			FOREIGN KEY(word_id) REFERENCES words(id) ON DELETE CASCADE,
			PRIMARY KEY(pattern, word_id)
		);
		`}

	// Note: FTS can't be applied on patterns because
	// we require partial word search which FTS doesn't support

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
		vals = append(vals, words[0])

		for i, word := range words {
			if i == 0 {
				continue
			}
			likes += ", (?)"
			vals = append(vals, word)
		}

		// Thanks forpas
		// CC BY-SA 4.0 licensed
		// https://stackoverflow.com/q/68610241/1372424

		if all == true {
			query = "WITH cte(match) AS (VALUES (?) " + likes + ") SELECT w.* FROM words_fts w INNER JOIN cte c ON w.word MATCH c.match || '*' AND w.word != c.match AND learned_on > 0 ORDER BY weight DESC LIMIT ?"
			vals = append(vals, varnam.DictionarySuggestionsLimit)
		} else {
			query = "WITH cte(match) AS (VALUES (?) " + likes + ") SELECT c.match AS word, MAX(w.weight), MAX(w.learned_on) FROM words_fts w INNER JOIN cte c ON w.word MATCH c.match || '*' GROUP BY c.match"
		}

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
					start := time.Now()
					var toSearch []string
					for _, possibility := range t.symbols {
						toSearch = append(toSearch, getSymbolValue(possibility, 0))
					}

					searchResults := varnam.searchDictionary(ctx, toSearch, false)

					tempFoundDictWords = searchResults
					results = searchResults

					if LOG_TIME_TAKEN {
						log.Printf("%s took %v\n", "getFromDictionaryToken0", time.Since(start))
					}
				} else {
					start := time.Now()
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
					if LOG_TIME_TAKEN {
						log.Printf("%s%d took %v\n", "getFromDictionaryToken", i, time.Since(start))
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
		rows, err := varnam.dictConn.QueryContext(ctx, "SELECT word, weight FROM words WHERE id IN (SELECT word_id FROM patterns WHERE pattern LIKE ?) ORDER BY weight DESC LIMIT 10", pattern+"%")

		if err != nil {
			log.Print(err)
			return results
		}

		defer rows.Close()

		for rows.Next() {
			var item Suggestion
			rows.Scan(&item.Word, &item.Weight)
			item.Weight += VARNAM_LEARNT_WORD_MIN_WEIGHT
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
		rows, err := varnam.dictConn.QueryContext(ctx, "SELECT LENGTH(pts.pattern), w.word, w.weight, w.learned_on FROM `patterns` pts LEFT JOIN words w ON w.id = pts.word_id WHERE ? LIKE (pts.pattern || '%') OR pattern LIKE ? ORDER BY LENGTH(pts.pattern) DESC LIMIT ?", pattern, pattern+"%", varnam.PatternDictionarySuggestionsLimit)

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
