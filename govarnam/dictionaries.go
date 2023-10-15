package govarnam

import (
	"context"
	sql "database/sql"
)

type DictionaryConfig struct {
	Path  string
	Write bool // Use for storing user learnt words
	conn  *sql.DB
}

func (varnam *Varnam) setDefaultDictionariesConfig() {
	systemDictPath, err := findSystemDictionaryPath(varnam.SchemeDetails.LangCode)

	if err == nil {
		varnam.DictsConfig = append(varnam.DictsConfig, DictionaryConfig{systemDictPath, false, nil})
	}

	userDictPath := findUserDictionaryPath(varnam.SchemeDetails.LangCode)

	varnam.DictsConfig = append(varnam.DictsConfig, DictionaryConfig{userDictPath, true, nil})
}

func (varnam *Varnam) InitDicts() error {
	for _, dictConfig := range varnam.DictsConfig {
		if dictConfig.Write {
			err := varnam.InitDict(&dictConfig)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (varnam *Varnam) getWritableDicts() []DictionaryConfig {
	var result []DictionaryConfig
	for _, dictConfig := range varnam.DictsConfig {
		if !dictConfig.Write {
			continue
		}
		result = append(result, dictConfig)
	}
	return result
}

// GetRecentlyLearntWords get recently learnt words
func (varnam *Varnam) GetRecentlyLearntWords(
	ctx context.Context,
	offset int,
	limit int,
) ([]Suggestion, error) {
	var result []Suggestion

	select {
	case <-ctx.Done():
		return result, nil
	default:
		for _, dictConfig := range varnam.getWritableDicts() {
			sugs, err := varnam.getRecentlyLearntWordsFromDict(ctx, dictConfig.conn, offset, limit)
			if err != nil {
				return nil, err
			}

			result = append(result, sugs...)
		}
	}

	return result, nil
}

// GetSuggestions get word suggestions from all dictionaries
func (varnam *Varnam) GetSuggestions(ctx context.Context, word string) []Suggestion {
	var sugs []Suggestion

	select {
	case <-ctx.Done():
		return sugs
	default:
		for _, dictConfig := range varnam.DictsConfig {
			sugs = append(
				sugs,
				convertSearchDictResultToSuggestions(
					varnam.searchDictionary(ctx, dictConfig.conn, []string{word}, searchStartingWith),
					true,
				)...,
			)
		}

		return SortSuggestions(sugs)
	}
}
