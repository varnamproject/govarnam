package govarnamgo

/* Golang bindings for govarnam library */

// #cgo LDFLAGS: -L${SRCDIR}/../ -lgovarnam
// #cgo CFLAGS: -I${SRCDIR}/../
// #include "libgovarnam.h"
// #include "stdlib.h"
import "C"

import (
	"context"
	"fmt"
	"unsafe"
)

// Config  config values
type Config struct {
	IndicDigits                bool
	DictionarySuggestionsLimit int
	TokenizerSuggestionsLimit  int
	TokenizerSuggestionsAlways bool
}

type VarnamHandle struct {
	connectionID C.int
}

// Suggestion suggestion
type Suggestion struct {
	Word      string
	Weight    int
	LearnedOn int
}

// TransliterationResult result
type TransliterationResult struct {
	ExactMatch            []Suggestion
	Suggestions           []Suggestion
	GreedyTokenized       []Suggestion
	DictionaryResultCount int
}

// Convert a C Suggestion to Go
func makeSuggestion(cSug *C.struct_Suggestion_t) Suggestion {
	var sug Suggestion
	sug.Word = C.GoString(cSug.Word)
	sug.Weight = int(cSug.Weight)
	sug.LearnedOn = int(cSug.LearnedOn)

	return sug
}

func makeGoTransliterationResult(ctx context.Context, cResult *C.struct_TransliterationResult_t) TransliterationResult {
	var result TransliterationResult

	select {
	case <-ctx.Done():
		return result
	default:
		var i int

		var exactMatch []Suggestion
		i = 0
		for i < int(C.varray_length(cResult.ExactMatch)) {
			cSug := (*C.Suggestion)(C.varray_get(cResult.ExactMatch, C.int(i)))
			sug := makeSuggestion(cSug)
			exactMatch = append(exactMatch, sug)
			i++
		}
		result.ExactMatch = exactMatch

		var suggestions []Suggestion
		i = 0
		for i < int(C.varray_length(cResult.Suggestions)) {
			cSug := (*C.Suggestion)(C.varray_get(cResult.Suggestions, C.int(i)))
			sug := makeSuggestion(cSug)
			suggestions = append(suggestions, sug)
			i++
		}
		result.Suggestions = suggestions

		var greedyTokenized []Suggestion
		i = 0
		for i < int(C.varray_length(cResult.GreedyTokenized)) {
			cSug := (*C.Suggestion)(C.varray_get(cResult.Suggestions, C.int(i)))
			sug := makeSuggestion(cSug)
			greedyTokenized = append(greedyTokenized, sug)
			i++
		}
		result.GreedyTokenized = greedyTokenized

		go C.destroyTransliterationResult(cResult)

		return result
	}
}

func checkError(code C.int) bool {
	if code == C.VARNAM_SUCCESS {
		return true
	}
	return false
}

// InitFromID Initialize
func InitFromID(id string) (*VarnamHandle, error) {
	handleID := C.int(0)
	cID := C.CString(id)
	err := C.varnam_init_from_id(cID, unsafe.Pointer(&handleID))
	C.free(unsafe.Pointer(cID))

	if err != C.VARNAM_SUCCESS {
		return nil, fmt.Errorf(C.GoString(C.varnam_get_last_error(handleID)))
	}
	return &VarnamHandle{handleID}, nil
}

// GetLastError get last error
func (handle *VarnamHandle) GetLastError() string {
	cStr := C.varnam_get_last_error(handle.connectionID)
	goStr := C.GoString(cStr)
	C.free(unsafe.Pointer(cStr))
	return goStr
}

// Debug turn debug on/off
func (handle *VarnamHandle) Debug(val bool) {
	if val {
		C.varnam_debug(handle.connectionID, C.int(1))
	} else {
		C.varnam_debug(handle.connectionID, C.int(0))
	}
}

// SetConfig set config
func (handle *VarnamHandle) SetConfig(config Config) {
	C.varnam_set_dictionary_suggestions_limit(handle.connectionID, C.int(config.DictionarySuggestionsLimit))

	C.varnam_set_tokenizer_suggestions_limit(handle.connectionID, C.int(config.TokenizerSuggestionsLimit))

	if config.IndicDigits {
		C.varnam_set_indic_digits(handle.connectionID, C.int(1))
	} else {
		C.varnam_set_indic_digits(handle.connectionID, C.int(0))
	}
}

func (handle *VarnamHandle) cgoGetTransliterationResult(operationID C.int, resultChannel chan<- *C.struct_TransliterationResult_t, word string) {
	cWord := C.CString(word)
	defer C.free(unsafe.Pointer(cWord))
	cResult := C.varnam_transliterate_with_id(handle.connectionID, operationID, cWord)

	resultChannel <- cResult
	close(resultChannel)
}

var transliterationOperationCount = C.int(0)

// Transliterate transilterate
func (handle *VarnamHandle) Transliterate(ctx context.Context, word string) TransliterationResult {
	operationID := transliterationOperationCount
	transliterationOperationCount++

	channel := make(chan *C.struct_TransliterationResult_t)

	go handle.cgoGetTransliterationResult(operationID, channel, word)

	select {
	case <-ctx.Done():
		C.varnam_cancel(operationID)
		var result TransliterationResult
		return result
	case cResult := <-channel:
		return makeGoTransliterationResult(ctx, cResult)
	}
}

// Train train a pattern => word
func (handle *VarnamHandle) Train(pattern string, word string) bool {
	cPattern := C.CString(pattern)
	cWord := C.CString(word)

	err := C.varnam_train(handle.connectionID, cPattern, cWord)

	C.free(unsafe.Pointer(cPattern))
	C.free(unsafe.Pointer(cWord))

	return checkError(err)
}

// Learn a word
func (handle *VarnamHandle) Learn(word string, confidence int) bool {
	cWord := C.CString(word)

	err := C.varnam_learn(handle.connectionID, cWord, C.int(confidence))

	C.free(unsafe.Pointer(cWord))

	return checkError(err)
}

// Unlearn a word
func (handle *VarnamHandle) Unlearn(word string) bool {
	cWord := C.CString(word)

	err := C.varnam_unlearn(handle.connectionID, cWord)

	C.free(unsafe.Pointer(cWord))

	return checkError(err)
}

// LearnFromFile learn words from a file
func (handle *VarnamHandle) LearnFromFile(filePath string) bool {
	cFilePath := C.CString(filePath)
	err := C.varnam_learn_from_file(handle.connectionID, cFilePath)
	return checkError(err)
}

// TrainFromFile train pattern => word from a file
func (handle *VarnamHandle) TrainFromFile(filePath string) bool {
	cFilePath := C.CString(filePath)
	err := C.varnam_train_from_file(handle.connectionID, cFilePath)
	return checkError(err)
}
