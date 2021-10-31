package govarnam

import (
	"context"
	sql "database/sql"
	"fmt"
	"time"
)

// vm = Vst Maker

func (varnam *Varnam) vmStartBuffering() error {
	if varnam.VSTMakerConfig.Buffering {
		return nil
	}

	varnam.vstConn.Exec("BEGIN;")
	varnam.VSTMakerConfig.Buffering = true
	return nil
}

func (varnam *Varnam) vmFlushChanges() error {
	if !varnam.VSTMakerConfig.Buffering {
		return nil
	}

	varnam.log("Writing changes to file...")
	_, err := varnam.vstConn.Exec("COMMIT;")
	if err != nil {
		return fmt.Errorf("failed to flush changes: " + err.Error())
	}

	varnam.VSTMakerConfig.Buffering = false

	varnam.log("Compacting file...")
	_, err = varnam.vstConn.Exec("VACUUM")
	if err != nil {
		return fmt.Errorf("failed to compact db: " + err.Error())
	}

	return nil
}

// This function is called when something went wrong. Rollback VST DB
func (varnam *Varnam) vmDiscardChanges() error {
	if !varnam.VSTMakerConfig.Buffering {
		return nil
	}

	varnam.vstConn.Exec("ROLLBACK;")
	varnam.VSTMakerConfig.Buffering = false
	return nil
}

// VMCreateToken Create Token
func (varnam *Varnam) VMCreateToken(pattern string, value1 string, value2 string, value3 string, tag string, symbolType int, matchType int, priority int, acceptCondition int, buffered bool) error {
	if pattern == "" || value1 == "" {
		return fmt.Errorf("pattern or value1 is empty")
	}

	if len(pattern) > VARNAM_SYMBOL_MAX || len(value1) > VARNAM_SYMBOL_MAX || (value2 != "" && len(value2) > VARNAM_SYMBOL_MAX) ||
		(value3 != "" && len(value3) > VARNAM_SYMBOL_MAX) ||
		(tag != "" && len(tag) > VARNAM_SYMBOL_MAX) {
		return fmt.Errorf("length of pattern, tag, value1 or value2, value3 should be less than VARNAM_SYMBOL_MAX")
	}

	if matchType != VARNAM_MATCH_EXACT && matchType != VARNAM_MATCH_POSSIBILITY {
		return fmt.Errorf("matchType should be either VARNAM_MATCH_EXACT or VARNAM_MATCH_POSSIBILITY")
	}

	if acceptCondition != VARNAM_TOKEN_ACCEPT_ALL &&
		acceptCondition != VARNAM_TOKEN_ACCEPT_IF_STARTS_WITH &&
		acceptCondition != VARNAM_TOKEN_ACCEPT_IF_IN_BETWEEN &&
		acceptCondition != VARNAM_TOKEN_ACCEPT_IF_ENDS_WITH {
		return fmt.Errorf("invalid accept condition specified. It should be one of VARNAM_TOKEN_ACCEPT_XXX")
	}

	if buffered {
		varnam.vmStartBuffering()
	}

	if symbolType == VARNAM_SYMBOL_CONSONANT && varnam.VSTMakerConfig.UseDeadConsonants {
		virama, err := varnam.getVirama()
		if err != nil {
			return fmt.Errorf("virama needs to be set before auto generating dead consonants")
		}

		patternRune := []rune(pattern)

		lastChar, _ := getLastCharacter(value1)
		if lastChar == virama {
			symbolType = VARNAM_SYMBOL_DEAD_CONSONANT
		} else if canGenerateDeadConsonant(patternRune) {
			patternExceptLastChar := string(patternRune[:len(patternRune)-1])

			var (
				value1WithVirama = value1 + virama
				value2WithVirama = value2
			)
			if value2 != "" {
				value2WithVirama = value2 + virama
			}

			err := varnam.vmPersistToken(patternExceptLastChar, value1WithVirama, value2WithVirama, value3, tag, VARNAM_SYMBOL_DEAD_CONSONANT, matchType, priority, acceptCondition)

			if err != nil {
				varnam.vmDiscardChanges()
				return err
			}
		}
	}

	if symbolType == VARNAM_SYMBOL_NON_JOINER {
		value1 = ZWNJ
		value2 = ZWNJ
	}

	if symbolType == VARNAM_SYMBOL_JOINER {
		value1 = ZWJ
		value2 = ZWJ
	}

	err := varnam.vmPersistToken(pattern, value1, value2, value3, tag, symbolType, matchType, priority, acceptCondition)
	if err != nil {
		if buffered {
			varnam.vmDiscardChanges()
		}
		return err
	}

	if !buffered {
		// TODO flags is not used in govarnam
		// err = varnam.vmMakePrefixTree()
		// if err != nil {
		// 	return err
		// }

		err = varnam.vmStampVersion()
		if err != nil {
			return err
		}
	}

	return nil
}

func (varnam *Varnam) vmPersistToken(pattern string, value1 string, value2 string, value3 string, tag string, symbolType int, matchType int, priority int, acceptCondition int) error {
	if pattern == "" || value1 == "" || !(symbolType >= VARNAM_SYMBOL_VOWEL && symbolType <= VARNAM_SYMBOL_PERIOD) {
		return fmt.Errorf("arguments invalid")
	}

	if varnam.vmAlreadyPersisted(pattern, value1, matchType) {
		// TODO the equivalent libvarnam is unclear what it was trying to do
		// if varnam.VSTMakerConfig.IgnoreDuplicateTokens {
		// 	varnam.log(fmt.Sprintf("%s => %s is already available. Ignoring duplicate tokens", pattern, value1))
		// 	return nil
		// }

		return fmt.Errorf("there is already a match available for '%s => %s'. Duplicate entries are not allowed", pattern, value1)
	}

	query := "INSERT OR IGNORE INTO symbols (type, pattern, value1, value2, value3, tag, matchType, priority, acceptCondition) VALUES (?, trim(?), trim(?), trim(?), trim(?), trim(?), ?, ?, ?)"

	bgContext := context.Background()

	ctx, cancelFunc := context.WithTimeout(bgContext, 5*time.Second)
	defer cancelFunc()

	stmt, err := varnam.dictConn.PrepareContext(ctx, query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, symbolType, pattern, value1, value2, value3, tag, matchType, priority, acceptCondition)
	if err != nil {
		return fmt.Errorf("Failed to persist token: %s", err.Error())
	}

	return nil
}

func (varnam *Varnam) vmAlreadyPersisted(pattern string, value1 string, matchType int) bool {
	var searchCriteria Symbol
	searchCriteria.Pattern = pattern
	searchCriteria.Value1 = value1
	searchCriteria.MatchType = matchType

	result, _ := varnam.SearchSymbolTable(context.Background(), searchCriteria)

	return len(result) > 0
}

// Makes a prefix tree. This fills up the flags column
// TODO incomplete
func (varnam *Varnam) vmMakePrefixTree() error {
	for _, columnName := range []string{"pattern", "value1", "value2"} {
		stmt, err := varnam.vstConn.Prepare(fmt.Sprintf("SELECT id, %s FROM symbols GROUP BY %s ORDER BY LENGTH(%s) ASC", columnName, columnName, columnName))

		if err != nil {
			varnam.log(err.Error())
			return nil
		}

		var mask int
		if columnName == "pattern" {
			mask = VARNAM_SYMBOL_FLAGS_MORE_MATCHES_FOR_PATTERN
		} else {
			mask = VARNAM_SYMBOL_FLAGS_MORE_MATCHES_FOR_VALUE
		}

		updateStmt, err := varnam.vstConn.Prepare(fmt.Sprintf("UPDATE symbols SET flags = flags | %d WHERE %s = ?", mask, columnName))
		if err != nil {
			varnam.log(err.Error())
		}

		varnam.vmFindPrefixesAndUpdateFlags(stmt, updateStmt)
		stmt.Close()
		updateStmt.Close()
	}

	return nil
}

func (varnam *Varnam) vmFindPrefixesAndUpdateFlags(stmt *sql.Stmt, updateStmt *sql.Stmt) {
	type symbolIDMap struct {
		symbol string
		id     int
	}
	// TODO incomplete
}

func (varnam *Varnam) vmStampVersion() error {
	_, err := varnam.vstConn.Exec(fmt.Sprintf("PRAGMA user_version=%d", VARNAM_SCHEMA_SYMBOLS_VERSION))
	return err
}

func (varnam *Varnam) vmAddMetadata(key string, value string) error {
	_, err := varnam.vstConn.Exec("INSERT OR REPLACE INTO metadata (key, value) VALUES (?, ?)", key, value)
	return err
}

// VMSetSchemeDetails set scheme details
func (varnam *Varnam) VMSetSchemeDetails(sd SchemeDetails) error {
	if len(sd.LangCode) != 2 {
		return fmt.Errorf("language code should be one of ISO 639-1 two letter codes")
	}

	type item struct {
		name  string
		key   string
		value string
	}

	items := []item{
		{"language code", VARNAM_METADATA_SCHEME_LANGUAGE_CODE, sd.LangCode},
		{"language identifier", VARNAM_METADATA_SCHEME_IDENTIFIER, sd.Identifier},
		{"language display name", VARNAM_METADATA_SCHEME_DISPLAY_NAME, sd.DisplayName},
		{"author", VARNAM_METADATA_SCHEME_AUTHOR, sd.Author},
		{"compiled date", VARNAM_METADATA_SCHEME_COMPILED_DATE, sd.CompiledDate},
	}

	for _, o := range items {
		err := varnam.vmAddMetadata(o.key, o.value)
		if err != nil {
			return err
		}
		varnam.log("Set " + o.name + " to: " + string(o.value))
	}

	return nil
}

// VMFlushBuffer flush
func (varnam *Varnam) VMFlushBuffer() error {
	// varnam.vmMakePrefixTree()

	err := varnam.vmStampVersion()
	if err != nil {
		return err
	}

	return varnam.vmFlushChanges()
}

// Checks if the string has inherent 'a' sound. If yes, we can infer dead consonant from it
func canGenerateDeadConsonant(input []rune) bool {
	if len(input) <= 1 {
		return false
	}
	return string(input[len(input)-2]) != "a" &&
		string(input[len(input)-1]) == "a"
}
