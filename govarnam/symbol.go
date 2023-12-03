package govarnam

/**
 * govarnam - An Indian language transliteration library
 * Copyright Subin Siby <mail at subinsb (.) com>, 2021
 * Licensed under AGPL-3.0-only. See LICENSE.txt
 */

import (
	"context"
	sql "database/sql"
	"fmt"
	"log"
	"strings"
	"unicode"

	"github.com/mattn/go-sqlite3"
)

// Symbol result from VST
type Symbol struct {
	Identifier      int
	Type            int
	MatchType       int
	Pattern         string
	Value1          string
	Value2          string
	Value3          string
	Tag             string
	Weight          int
	Priority        int
	AcceptCondition int
	Flags           int
}

// Token info for making a suggestion
type Token struct {
	tokenType int
	symbols   []Symbol // Will be empty for non language character
	position  int
	character string // Non language character
}

var sqlite3WithLimitDriverRegistered bool
var sqlite3Conn *sqlite3.SQLiteConn

func openDB(path string) (*sql.DB, error) {
	if !sqlite3WithLimitDriverRegistered {
		sql.Register("sqlite3_with_limit", &sqlite3.SQLiteDriver{
			ConnectHook: func(conn *sqlite3.SQLiteConn) error {
				sqlite3Conn = conn
				return nil
			},
		})
		sqlite3WithLimitDriverRegistered = true
	}

	conn, err := sql.Open("sqlite3_with_limit", path)
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

	err = varnam.setPatternLongestLength()
	if err != nil {
		return err
	}

	varnam.vstConn.Exec("PRAGMA TEMP_STORE=2;")
	varnam.vstConn.Exec("PRAGMA LOCKING_MODE=EXCLUSIVE;")

	varnam.VSTPath = vstPath
	varnam.setSchemeInfo()

	return nil
}

// Find the longest pattern length
func (varnam *Varnam) setPatternLongestLength() error {
	rows, err := varnam.vstConn.Query("SELECT MAX(LENGTH(pattern)) FROM symbols")
	if err != nil {
		return err
	}

	length := 0
	for rows.Next() {
		err := rows.Scan(&length)
		if err != nil {
			return err
		}
	}

	if length == 0 {
		return fmt.Errorf("couldn't find longest pattern length")
	}

	varnam.LangRules.PatternLongestLength = length

	return nil
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
			varnam.SchemeDetails.Identifier = value
		} else if key == "lang-code" {
			varnam.SchemeDetails.LangCode = value
		} else if key == "scheme-display-name" {
			varnam.SchemeDetails.DisplayName = value
		} else if key == "scheme-author" {
			varnam.SchemeDetails.Author = value
		} else if key == "scheme-compiled-date" {
			varnam.SchemeDetails.CompiledDate = value
		} else if key == "scheme-stable" {
			if value == "1" {
				varnam.SchemeDetails.IsStable = true
			} else {
				varnam.SchemeDetails.IsStable = false
			}
		}
	}
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
			rows, err = varnam.vstConn.QueryContext(ctx, "SELECT * FROM symbols WHERE (value1 = ? OR value2 = ?) AND (accept_condition = 0 OR accept_condition = ?) ORDER BY match_type ASC, weight DESC, priority DESC", ch, ch, acceptCondition)
		} else {
			rows, err = varnam.vstConn.QueryContext(ctx, "SELECT * FROM symbols WHERE (value1 = ? OR value2 = ?) AND match_type = ? AND (accept_condition = 0 OR accept_condition = ?)", ch, ch, matchType, acceptCondition)
		}

		if err != nil {
			log.Print(err)
			return results
		}
		defer rows.Close()

		for rows.Next() {
			var item Symbol
			rows.Scan(&item.Identifier, &item.Type, &item.Pattern, &item.Value1, &item.Value2, &item.Value3, &item.Tag, &item.MatchType, &item.Priority, &item.AcceptCondition, &item.Flags, &item.Weight)
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
			query = "SELECT * FROM `symbols` WHERE (accept_condition = 0 OR accept_condition = ?) AND pattern IN (? " + patternINs + ") ORDER BY LENGTH(pattern) DESC, match_type ASC, weight DESC, priority DESC"
		} else {
			query = "SELECT * FROM `symbols` WHERE match_type = ? AND (accept_condition = 0 OR accept_condition = ?) AND pattern IN (? " + patternINs + ") ORDER BY LENGTH(pattern) DESC"
		}

		rows, err := varnam.vstConn.QueryContext(ctx, query, vals...)

		if err != nil {
			log.Print(err)
			return results
		}
		defer rows.Close()

		for rows.Next() {
			var item Symbol
			rows.Scan(&item.Identifier, &item.Type, &item.Pattern, &item.Value1, &item.Value2, &item.Value3, &item.Tag, &item.MatchType, &item.Priority, &item.AcceptCondition, &item.Flags, &item.Weight)
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
				if unicode.In(sequence[0], &varnam.LangRules.UnicodeBlock) {
					// This helps to get suggestions in inputs like "ആലppu"
					character := string(sequence[0])
					token := Token{VARNAM_TOKEN_SYMBOL, []Symbol{{Value1: character}}, i, character}
					results = append(results, token)
				} else {
					// No matches, add a character token
					// Note that we just add 1 character, and move on
					token := Token{VARNAM_TOKEN_CHAR, matches, i, string(sequence[:1])}
					results = append(results, token)
				}

				i++
			} else {
				if matches[0].Type == VARNAM_SYMBOL_NUMBER && !varnam.LangRules.IndicDigits {
					// Skip numbers
					// Note that we just add 1 character, and move on
					token := Token{VARNAM_TOKEN_CHAR, []Symbol{}, i, string(sequence[:1])}
					results = append(results, token)

					i += len(matches[0].Pattern)
				} else {
					// Add matches
					var refinedMatches []Symbol
					longestPatternLength := 0

					for _, match := range matches {
						if longestPatternLength == 0 {
							// Sort is by length of pattern, so we will get length from first iterations.
							longestPatternLength = len(match.Pattern)
							refinedMatches = append(refinedMatches, match)
						} else {
							if len(match.Pattern) != longestPatternLength {
								break
							}
							refinedMatches = append(refinedMatches, match)
						}
					}
					i += longestPatternLength

					token := Token{VARNAM_TOKEN_SYMBOL, refinedMatches, i - 1, string(refinedMatches[0].Pattern)}
					results = append(results, token)
				}
			}
		}
		return &results
	}
}

// Tokenize end part of a word and append it to results
func (varnam *Varnam) tokenizeRestOfWord(ctx context.Context, word string, sugs []Suggestion, limit int) []Suggestion {
	var results []Suggestion

	if varnam.Debug {
		fmt.Printf("Tokenizing %s\n", word)
	}

	tokensPointerChan := make(chan *[]Token)
	go varnam.channelTokenizeWord(ctx, word, VARNAM_MATCH_ALL, true, tokensPointerChan)

	select {
	case <-ctx.Done():
		return results
	case restOfWordTokens := <-tokensPointerChan:
		for _, sug := range sugs {
			sugWord := varnam.removeLastVirama(sug.Word)
			tokensWithWord := []Token{{VARNAM_TOKEN_CHAR, []Symbol{}, 0, sugWord}}
			tokensWithWord = append(tokensWithWord, *restOfWordTokens...)

			restOfWordSugs := varnam.tokensToSuggestions(ctx, &tokensWithWord, true, limit)

			if varnam.Debug {
				fmt.Println("Tokenized & Added:", restOfWordSugs)
			}

			for _, restOfWordSug := range restOfWordSugs {
				// Preserve original word's weight and timestamp
				restOfWordSug.Weight += sug.Weight
				restOfWordSug.LearnedOn = sug.LearnedOn
				results = append(results, restOfWordSug)
			}
		}

		return results
	}
}

// Split an input string into tokens of symbols (conjuncts) and characters
func (varnam *Varnam) splitTextByConjunct(ctx context.Context, inputStr string) []Token {
	var results []Token

	var prevSequence string
	var prevSequenceMatches []Symbol

	var sequence string

	// Not using len() because it will be wrong for non ASCII characters
	var sequenceLength int

	input := []rune(inputStr)

	position := 0
	i := 0
	for i < len(input) {
		ch := string(input[i])

		sequence += ch
		sequenceLength++

		acceptCondition := VARNAM_TOKEN_ACCEPT_IF_IN_BETWEEN

		if i == 0 {
			// Trying to make the first token
			acceptCondition = VARNAM_TOKEN_ACCEPT_IF_STARTS_WITH
		} else if i == len(input)-1 {
			acceptCondition = VARNAM_TOKEN_ACCEPT_IF_ENDS_WITH
		}

		symbols := varnam.searchPattern(ctx, sequence, VARNAM_MATCH_ALL, acceptCondition)

		if len(symbols) == 0 {
			// No more matches

			if sequenceLength == 1 {
				// Has non language characters, add char token
				results = append(results, Token{VARNAM_TOKEN_CHAR, []Symbol{}, position, sequence})
			} else if len(prevSequenceMatches) > 0 {
				// Backtrack and add the previous sequence matches
				i--
				results = append(results, Token{VARNAM_TOKEN_SYMBOL, prevSequenceMatches, position, prevSequence})
			}

			sequence = ""
			sequenceLength = 0
			position++
		} else {
			if i == len(input)-1 {
				// Last character
				results = append(results, Token{VARNAM_TOKEN_SYMBOL, symbols, position, sequence})
				position++
			} else {
				prevSequence = sequence
				prevSequenceMatches = symbols
			}
		}
		i++
	}

	return results
}

// Split a word by conjuncts. Returns string of conjuncts of first full word found
func (varnam *Varnam) splitWordByConjunct(word string) []string {
	ctx := context.Background()
	var result []string
	tokens := varnam.splitTextByConjunct(ctx, word)

	if varnam.Debug {
		log.Println(tokens)
	}

	for _, token := range tokens {
		if token.tokenType == VARNAM_TOKEN_SYMBOL {
			ok := true

			for _, symbol := range token.symbols {
				if symbol.Type == VARNAM_SYMBOL_NUMBER || symbol.Type == VARNAM_SYMBOL_PERIOD || symbol.Type == VARNAM_SYMBOL_SYMBOL {
					ok = false
					break
				}
			}

			if ok {
				result = append(result, token.character)
			}
		} else {
			break
		}
	}
	return result
}

func getSymbolValue(symbol Symbol, position int) string {
	// Ignore render_value2 tag. It's only applicable for libvarnam
	// https://gitlab.com/subins2000/govarnam/-/issues/3

	if symbol.Type == VARNAM_SYMBOL_VOWEL && position > 0 {
		// If in between word, we use the vowel and not the consonant
		return symbol.Value2 // ാ
	}
	return symbol.Value1 // ആ
}

func getSymbolWeight(symbol Symbol) int {
	if symbol.MatchType == VARNAM_MATCH_EXACT {
		// 200 because there might be possibility matches having weight 100
		return 200
	}
	return symbol.Weight
}

// Removes less weighted symbols
func removeLessWeightedSymbols(tokens []Token) []Token {
	for i := range tokens {
		var reducedSymbols []Symbol
		for _, symbol := range tokens[i].symbols {
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
				if symbol.MatchType == VARNAM_MATCH_EXACT {
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

// NewSearchSymbol a constructor for making Symbol.
// We're doing this because default int value in
// go structs is 0. This won't work with searching
// because fields can have 0 value.
// https://stackoverflow.com/q/37135193/137242
func NewSearchSymbol() Symbol {
	symbol := Symbol{}
	symbol.Identifier = STRUCT_INT_DEFAULT_VALUE
	symbol.Type = STRUCT_INT_DEFAULT_VALUE
	symbol.MatchType = STRUCT_INT_DEFAULT_VALUE
	symbol.Weight = STRUCT_INT_DEFAULT_VALUE
	symbol.Priority = STRUCT_INT_DEFAULT_VALUE
	symbol.AcceptCondition = STRUCT_INT_DEFAULT_VALUE
	symbol.Flags = STRUCT_INT_DEFAULT_VALUE
	return symbol
}

func (varnam *Varnam) makeSearchSymbolQuery(queryPrefix string, searchCriteria Symbol) (string, []interface{}) {
	var (
		clauses []string
		values  []interface{}
	)

	addItem := func(name string, val interface{}) {
		if valString, isString := val.(string); isString {
			if valString == "" {
				return
			}

			// Format should be LIKE value
			if len(valString) > 5 && valString[0:5] == "LIKE " {
				valString = valString[5:]
				clauses = append(clauses, name+" LIKE ?")
				values = append(values, valString)

				return
			}
		} else {
			if valInt, ok := val.(int); !ok || valInt == STRUCT_INT_DEFAULT_VALUE {
				return
			}
		}
		clauses = append(clauses, name+" = ?")
		values = append(values, val)
	}

	addItem("id", searchCriteria.Identifier)
	addItem("type", searchCriteria.Type)
	addItem("match_type", searchCriteria.MatchType)
	addItem("pattern", searchCriteria.Pattern)
	addItem("value1", searchCriteria.Value1)
	addItem("value2", searchCriteria.Value2)
	addItem("value3", searchCriteria.Value3)
	addItem("tag", searchCriteria.Tag)
	addItem("weight", searchCriteria.Weight)
	addItem("priority", searchCriteria.Priority)
	addItem("accept_condition", searchCriteria.AcceptCondition)
	addItem("flags", searchCriteria.Flags)

	query := queryPrefix

	if len(values) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}

	if varnam.Debug {
		fmt.Println(query, values)
	}

	return query, values
}

// SearchSymbolTable For searching symbol table
func (varnam *Varnam) SearchSymbolTable(ctx context.Context, searchCriteria Symbol) ([]Symbol, error) {
	var results []Symbol

	select {
	case <-ctx.Done():
		return results, nil
	default:
		query, values := varnam.makeSearchSymbolQuery("SELECT * FROM symbols", searchCriteria)
		rows, err := varnam.vstConn.QueryContext(ctx, query, values...)
		if err != nil {
			return nil, err
		}

		for rows.Next() {
			var item Symbol
			rows.Scan(&item.Identifier, &item.Type, &item.Pattern, &item.Value1, &item.Value2, &item.Value3, &item.Tag, &item.MatchType, &item.Priority, &item.AcceptCondition, &item.Flags, &item.Weight)
			results = append(results, item)
		}

		return results, nil
	}
}

func (varnam *Varnam) getVirama() (string, error) {
	viramaSymbol := NewSearchSymbol()
	viramaSymbol.Pattern = "~"
	results, _ := varnam.SearchSymbolTable(context.Background(), viramaSymbol)

	if len(results) == 0 {
		return "", fmt.Errorf("virama not found")
	}

	return results[0].Value1, nil
}
