package govarnam

import (
	"context"
	sql "database/sql"
	"fmt"
	"log"
)

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
	weight          int
	priority        int
	acceptCondition int
	flags           int
}

// Token info for making a suggestion
type Token struct {
	tokenType int
	symbols   []Symbol // Will be empty for non language character
	position  int
	character string // Non language character
}

func openDB(path string) (*sql.DB, error) {
	conn, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

// InitVST initialize
func (varnam *Varnam) InitVST(vstPath string) error {
	var err error
	varnam.vstConn, err = openDB(vstPath + "?_case_sensitive_like=on")

	if err != nil {
		return err
	}

	varnam.setSchemeInfo()
	varnam.setPatternLongestLength()

	return nil
}

// Find the longest pattern length
func (varnam *Varnam) setPatternLongestLength() {
	rows, err := varnam.vstConn.Query("SELECT MAX(LENGTH(pattern)) FROM symbols")
	if err != nil {
		log.Print(err)
	}

	length := 0
	for rows.Next() {
		err := rows.Scan(&length)
		if err != nil {
			log.Print(err)
		}
	}
	varnam.LangRules.PatternLongestLength = length
}

func (varnam *Varnam) setSchemeInfo() {
	rows, err := varnam.vstConn.Query("SELECT * FROM metadata")

	if err != nil {
		log.Print(err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			key   string
			value string
		)
		rows.Scan(&key, &value)
		if key == "scheme-id" {
			varnam.SchemeInfo.SchemeID = value
		} else if key == "lang-code" {
			varnam.SchemeInfo.LangCode = value
		} else if key == "scheme-display-name" {
			varnam.SchemeInfo.DisplayName = value
		} else if key == "scheme-author" {
			varnam.SchemeInfo.Author = value
		} else if key == "scheme-compiled-date" {
			varnam.SchemeInfo.CompiledDate = value
		}
	}
}

// Checks if a symbol exist in VST
func (varnam *Varnam) symbolExist(ch string) (bool, error) {
	rows, err := varnam.vstConn.Query("SELECT COUNT(*) FROM symbols WHERE value1 = ? OR value2 = ? OR value3 = ?", ch, ch, ch)
	if err != nil {
		return false, err
	}

	count := 0
	for rows.Next() {
		err := rows.Scan(&count)
		if err != nil {
			return false, err
		}
	}
	return count != 0, nil
}

func (varnam *Varnam) searchPattern(ctx context.Context, ch string, matchType int, acceptCondition int) []Symbol {
	var (
		rows    *sql.Rows
		err     error
		results []Symbol
	)

	select {
	case <-ctx.Done():
		return results
	default:
		if matchType == VARNAM_MATCH_ALL {
			rows, err = varnam.vstConn.QueryContext(ctx, "SELECT id, type, match_type, pattern, value1, value2, value3, tag, weight, priority, accept_condition, flags from symbols WHERE value1 = ? AND (accept_condition = 0 OR accept_condition = ?) ORDER BY match_type ASC, weight DESC, priority DESC", ch, acceptCondition)
		} else {
			rows, err = varnam.vstConn.QueryContext(ctx, "SELECT id, type, match_type, pattern, value1, value2, value3, tag, weight, priority, accept_condition, flags from symbols WHERE value1 = ? AND match_type = ? AND (accept_condition = 0 OR accept_condition = ?)", ch, matchType, acceptCondition)
		}

		if err != nil {
			log.Print(err)
			return results
		}
		defer rows.Close()

		for rows.Next() {
			var item Symbol
			rows.Scan(&item.id, &item.generalType, &item.matchType, &item.pattern, &item.value1, &item.value2, &item.value3, &item.tag, &item.weight, &item.priority, &item.acceptCondition, &item.flags)
			results = append(results, item)
		}

		err = rows.Err()
		if err != nil {
			log.Print(err)
		}

		return results
	}
}

func (varnam *Varnam) searchSymbol(ctx context.Context, ch string, matchType int, acceptCondition int) []Symbol {
	var (
		rows    *sql.Rows
		err     error
		results []Symbol
	)

	select {
	case <-ctx.Done():
		return results
	default:
		if matchType == VARNAM_MATCH_ALL {
			rows, err = varnam.vstConn.QueryContext(ctx, "SELECT id, type, match_type, pattern, value1, value2, value3, tag, weight, priority, accept_condition, flags from symbols WHERE pattern = ? AND (accept_condition = 0 OR accept_condition = ?) ORDER BY match_type ASC, weight DESC, priority DESC", ch, acceptCondition)
		} else {
			rows, err = varnam.vstConn.QueryContext(ctx, "SELECT id, type, match_type, pattern, value1, value2, value3, tag, weight, priority, accept_condition, flags from symbols WHERE pattern = ? AND match_type = ? AND (accept_condition = 0 OR accept_condition = ?)", ch, matchType, acceptCondition)
		}

		if err != nil {
			log.Print(err)
			return results
		}
		defer rows.Close()

		for rows.Next() {
			var item Symbol
			rows.Scan(&item.id, &item.generalType, &item.matchType, &item.pattern, &item.value1, &item.value2, &item.value3, &item.tag, &item.weight, &item.priority, &item.acceptCondition, &item.flags)
			results = append(results, item)
		}

		err = rows.Err()
		if err != nil {
			log.Print(err)
		}

		return results
	}
}

// Find longest pattern prefix matching symbols from VST
func (varnam *Varnam) findLongestPatternMatchSymbols(ctx context.Context, pattern []rune, matchType int, acceptCondition int) []Symbol {
	var (
		query      string
		results    []Symbol
		patternINs string
		vals       []interface{}
	)

	if matchType != VARNAM_MATCH_ALL {
		vals = append(vals, matchType)
	}

	vals = append(vals, acceptCondition)
	vals = append(vals, string(pattern[0]))

	for i := range pattern {
		if i == 0 {
			continue
		}
		patternINs += ", ?"
		vals = append(vals, string(pattern[0:i+1]))
	}

	if varnam.Debug {
		// The query will be made like :
		//   SELECT * FROM symbols WHERE pattern IN ('e', 'en', 'ent', 'enth', 'entho')
		// Will fetch the longest prefix match
		// Idea from https://stackoverflow.com/a/1860279/1372424
		fmt.Println(patternINs, vals)
	}

	select {
	case <-ctx.Done():
		return results
	default:
		if matchType == VARNAM_MATCH_ALL {
			query = "SELECT id, type, match_type, pattern, value1, value2, value3, tag, weight, priority, accept_condition, flags FROM `symbols` WHERE (accept_condition = 0 OR accept_condition = ?) AND pattern IN (? " + patternINs + ") ORDER BY LENGTH(pattern) DESC, match_type ASC, weight DESC, priority DESC"
		} else {
			query = "SELECT id, type, match_type, pattern, value1, value2, value3, tag, weight, priority, accept_condition, flags FROM `symbols` WHERE match_type = ? AND (accept_condition = 0 OR accept_condition = ?) AND pattern IN (? " + patternINs + ") ORDER BY LENGTH(pattern) DESC"
		}

		rows, err := varnam.vstConn.QueryContext(ctx, query, vals...)

		if err != nil {
			log.Print(err)
			return results
		}
		defer rows.Close()

		for rows.Next() {
			var item Symbol
			rows.Scan(&item.id, &item.generalType, &item.matchType, &item.pattern, &item.value1, &item.value2, &item.value3, &item.tag, &item.weight, &item.priority, &item.acceptCondition, &item.flags)
			results = append(results, item)
		}

		err = rows.Err()
		if err != nil {
			log.Print(err)
		}

		return results
	}
}

// Convert a string into Tokens for later processing
func (varnam *Varnam) tokenizeWord(ctx context.Context, word string, matchType int, partial bool) *[]Token {
	var results []Token

	select {
	case <-ctx.Done():
		return &results
	default:
		runes := []rune(word)

		i := 0
		for i < len(runes) {
			end := i + varnam.LangRules.PatternLongestLength
			if len(runes) < end {
				end = len(runes)
			}
			// Get characters after 'i'th position
			sequence := runes[i:end]

			acceptCondition := VARNAM_TOKEN_ACCEPT_IF_IN_BETWEEN

			if len(results) == 0 && !partial {
				// Trying to make the first token
				acceptCondition = VARNAM_TOKEN_ACCEPT_IF_STARTS_WITH
			} else if i == len(runes)-1 {
				acceptCondition = VARNAM_TOKEN_ACCEPT_IF_ENDS_WITH
			}

			matches := varnam.findLongestPatternMatchSymbols(ctx, sequence, matchType, acceptCondition)

			if len(matches) == 0 {
				// No matches, add a character token
				// Note that we just add 1 character, and move on
				token := Token{VARNAM_TOKEN_CHAR, matches, i, string(sequence[:1])}
				results = append(results, token)

				i++
			} else {
				if matches[0].generalType == VARNAM_SYMBOL_NUMBER && !varnam.LangRules.IndicDigits {
					// Skip numbers
					token := Token{VARNAM_TOKEN_CHAR, []Symbol{}, i, string(sequence)}
					results = append(results, token)

					i += len(matches[0].pattern)
				} else {
					// Add matches
					var refinedMatches []Symbol
					longestPatternLength := 0

					for _, match := range matches {
						if longestPatternLength == 0 {
							// Sort is by length of pattern, so we will get length from first iterations.
							longestPatternLength = len(match.pattern)
							refinedMatches = append(refinedMatches, match)
						} else {
							if len(match.pattern) != longestPatternLength {
								break
							}
							refinedMatches = append(refinedMatches, match)
						}
					}
					i += longestPatternLength

					token := Token{VARNAM_TOKEN_SYMBOL, refinedMatches, i - 1, string(refinedMatches[0].pattern)}
					results = append(results, token)
				}
			}
		}
		return &results
	}
}

// Tokenize end part of a word and append it to results
func (varnam *Varnam) tokenizeRestOfWord(ctx context.Context, word string, results []Suggestion) []Suggestion {
	select {
	case <-ctx.Done():
		return results
	default:
		if varnam.Debug {
			fmt.Printf("Tokenizing %s\n", word)
		}

		restOfWordTokens := varnam.tokenizeWord(ctx, word, VARNAM_MATCH_ALL, true)
		restOfWordSugs := varnam.tokensToSuggestions(ctx, restOfWordTokens, true)

		if varnam.Debug {
			fmt.Println("Tokenized:", restOfWordSugs)
		}

		if len(restOfWordSugs) > 0 {
			for j, result := range results {
				till := varnam.removeLastVirama(result.Word)
				tillWeight := result.Weight
				tillLearnedOn := result.LearnedOn

				firstSug := restOfWordSugs[0]
				results[j].Word = varnam.removeLastVirama(results[j].Word) + firstSug.Word
				results[j].Weight += firstSug.Weight

				for k, sug := range restOfWordSugs {
					if k == 0 {
						continue
					}
					sug := Suggestion{till + sug.Word, tillWeight + sug.Weight, tillLearnedOn}
					results = append(results, sug)
				}
			}
		}

		return results
	}
}

// Split a word into conjuncts
func (varnam *Varnam) splitWordByConjunct(input string) ([]string, error) {
	var results []string

	var prevSequenceMatch string
	var sequence string

	// Not using len() because it will be wrong for non ASCII characters
	var sequenceLength int

	word := []rune(input)

	i := 0
	for i < len(word) {
		ch := string(word[i])

		sequence += ch
		sequenceLength++

		doesntExist, err := varnam.symbolExist(sequence)
		if err != nil {
			return results, err
		}

		if !doesntExist {
			// No more matches

			if sequenceLength == 1 {
				// Has non language characters, give error
				return []string{}, fmt.Errorf("Has non language characters: %s", sequence)
			} else if len(prevSequenceMatch) > 0 {
				// Backtrack and add the previous sequence matches
				i--
				results = append(results, prevSequenceMatch)
			}

			sequence = ""
			sequenceLength = 0
		} else {
			if i == len(word)-1 {
				// Last character
				results = append(results, sequence)
			} else {
				prevSequenceMatch = sequence
			}
		}
		i++
	}
	return results, nil
}

func getSymbolValue(symbol Symbol, position int) string {
	// Ignore render_value2 tag. It's only applicable for libvarnam
	// https://gitlab.com/subins2000/govarnam/-/issues/3

	if symbol.generalType == VARNAM_SYMBOL_VOWEL && position > 0 {
		// If in between word, we use the vowel and not the consonant
		return symbol.value2 // ാ
	}
	return symbol.value1 // ആ
}

func getSymbolWeight(symbol Symbol) int {
	if symbol.matchType == VARNAM_MATCH_EXACT {
		// 200 because there might be possibility matches having weight 100
		return 200
	}
	return symbol.weight
}

// Removes less weighted symbols
func removeLessWeightedSymbols(tokens []Token) []Token {
	for i, token := range tokens {
		var reducedSymbols []Symbol
		for _, symbol := range token.symbols {
			// TODO should 0 be fixed for all languages ?
			// Because this may differ according to data source
			// from where symbol frequency was found out
			if getSymbolWeight(symbol) == 0 && len(reducedSymbols) > 0 {
				break
			}
			reducedSymbols = append(reducedSymbols, symbol)
		}
		tokens[i].symbols = nil
		tokens[i].symbols = reducedSymbols
	}
	return tokens
}

// Remove non-exact matching tokens
func removeNonExactTokens(tokens []Token) []Token {
	// Remove non-exact symbols
	for i, token := range tokens {
		if token.tokenType == VARNAM_TOKEN_SYMBOL {
			var reducedSymbols []Symbol
			for _, symbol := range token.symbols {
				if symbol.matchType == VARNAM_MATCH_EXACT {
					reducedSymbols = append(reducedSymbols, symbol)
				} else {
					if len(reducedSymbols) == 0 {
						// No exact matches, so add the first possibility match
						reducedSymbols = append(reducedSymbols, symbol)
					}
					// If a possibility result, then rest of them will also be same
					// so save time by skipping rest
					break
				}
			}
			tokens[i].symbols = reducedSymbols
		}
	}
	return tokens
}
