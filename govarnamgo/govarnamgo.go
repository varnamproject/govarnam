package govarnamgo

/**
 * govarnam - An Indian language transliteration library
 * Copyright Subin Siby <mail at subinsb (.) com>, 2021
 * Licensed under AGPL-3.0-only. See LICENSE.txt
 */

/* Golang bindings for govarnam library */

// #cgo pkg-config: govarnam
// #include "libgovarnam.h"
// #include "stdlib.h"
import "C"

import (
	"context"
	"fmt"
	"log"
	"unsafe"
)

// Config  config values
type Config struct {
	IndicDigits                       bool
	DictionaryMatchExact              bool
	DictionarySuggestionsLimit        int
	PatternDictionarySuggestionsLimit int
	TokenizerSuggestionsLimit         int
	TokenizerSuggestionsAlways        bool
}

// VarnamHandle for making things easier
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
	ExactWords                   []Suggestion
	ExactMatches                 []Suggestion
	DictionarySuggestions        []Suggestion
	PatternDictionarySuggestions []Suggestion
	TokenizerSuggestions         []Suggestion
	GreedyTokenized              []Suggestion
}

// SchemeDetails of VST
type SchemeDetails struct {
	Identifier   string
	LangCode     string
	DisplayName  string
	Author       string
	CompiledDate string
	IsStable     bool
}

// LearnStatus output of bulk learn
type LearnStatus struct {
	TotalWords  int
	FailedWords int
}

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

var contextOperationCount = C.int(0)

func makeContextOperation() C.int {
	operationID := contextOperationCount
	contextOperationCount++

	return operationID
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

		var exactWords []Suggestion
		i = 0
		for i < int(C.varray_length(cResult.ExactWords)) {
			cSug := (*C.Suggestion)(C.varray_get(cResult.ExactWords, C.int(i)))
			sug := makeSuggestion(cSug)
			exactWords = append(exactWords, sug)
			i++
		}
		result.ExactWords = exactWords

		var exactMatches []Suggestion
		i = 0
		for i < int(C.varray_length(cResult.ExactMatches)) {
			cSug := (*C.Suggestion)(C.varray_get(cResult.ExactMatches, C.int(i)))
			sug := makeSuggestion(cSug)
			exactMatches = append(exactMatches, sug)
			i++
		}
		result.ExactMatches = exactMatches

		var dictionarySuggestions []Suggestion
		i = 0
		for i < int(C.varray_length(cResult.DictionarySuggestions)) {
			cSug := (*C.Suggestion)(C.varray_get(cResult.DictionarySuggestions, C.int(i)))
			sug := makeSuggestion(cSug)
			dictionarySuggestions = append(dictionarySuggestions, sug)
			i++
		}
		result.DictionarySuggestions = dictionarySuggestions

		var patternDictionarySuggestions []Suggestion
		i = 0
		for i < int(C.varray_length(cResult.PatternDictionarySuggestions)) {
			cSug := (*C.Suggestion)(C.varray_get(cResult.PatternDictionarySuggestions, C.int(i)))
			sug := makeSuggestion(cSug)
			patternDictionarySuggestions = append(patternDictionarySuggestions, sug)
			i++
		}
		result.PatternDictionarySuggestions = patternDictionarySuggestions

		var tokenizerSuggestions []Suggestion
		i = 0
		for i < int(C.varray_length(cResult.TokenizerSuggestions)) {
			cSug := (*C.Suggestion)(C.varray_get(cResult.TokenizerSuggestions, C.int(i)))
			sug := makeSuggestion(cSug)
			tokenizerSuggestions = append(tokenizerSuggestions, sug)
			i++
		}
		result.TokenizerSuggestions = tokenizerSuggestions

		var greedyTokenized []Suggestion
		i = 0
		for i < int(C.varray_length(cResult.GreedyTokenized)) {
			cSug := (*C.Suggestion)(C.varray_get(cResult.GreedyTokenized, C.int(i)))
			sug := makeSuggestion(cSug)
			greedyTokenized = append(greedyTokenized, sug)
			i++
		}
		result.GreedyTokenized = greedyTokenized

		go C.destroyTransliterationResult(cResult)

		return result
	}
}

//VarnamError Custom error for varnam
type VarnamError struct {
	ErrorCode int
	Message   string
}

// Error mimicking error package's function
func (err VarnamError) Error() string {
	return err.Message
}

func (handle *VarnamHandle) checkError(code C.int) error {
	if code == C.VARNAM_SUCCESS {
		return nil
	}
	return &VarnamError{
		ErrorCode: int(code),
		Message:   handle.GetLastError(),
	}
}

// GetVersion get library version
func GetVersion() string {
	return C.GoString(C.varnam_get_version())
}

// GetBuild get library build version
func GetBuild() string {
	return C.GoString(C.varnam_get_build())
}

// Init Initialize
func Init(vstLoc string, dictLoc string) (*VarnamHandle, error) {
	handleID := C.int(0)
	cVSTFile := C.CString(vstLoc)
	cDictLoc := C.CString(dictLoc)

	err := C.varnam_init(cVSTFile, cDictLoc, unsafe.Pointer(&handleID))

	C.free(unsafe.Pointer(cVSTFile))
	C.free(unsafe.Pointer(cDictLoc))

	if err != C.VARNAM_SUCCESS {
		return nil, fmt.Errorf(C.GoString(C.varnam_get_last_error(handleID)))
	}
	return &VarnamHandle{handleID}, nil
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

// Close db connections and end varnam
func (handle *VarnamHandle) Close() error {
	err := C.varnam_close(handle.connectionID)
	return handle.checkError(err)
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

	C.varnam_set_pattern_dictionary_suggestions_limit(handle.connectionID, C.int(config.PatternDictionarySuggestionsLimit))

	C.varnam_set_tokenizer_suggestions_limit(handle.connectionID, C.int(config.TokenizerSuggestionsLimit))

	if config.IndicDigits {
		C.varnam_set_indic_digits(handle.connectionID, C.int(1))
	} else {
		C.varnam_set_indic_digits(handle.connectionID, C.int(0))
	}

	if config.DictionaryMatchExact {
		C.varnam_set_dictionary_match_exact(handle.connectionID, C.int(1))
	} else {
		C.varnam_set_dictionary_match_exact(handle.connectionID, C.int(0))
	}
}

type cgoVarnamTransliterateResult struct {
	result *C.varray
	err    error
}

func (handle *VarnamHandle) cgoVarnamTransliterate(operationID C.int, resultChannel chan<- cgoVarnamTransliterateResult, word string) {
	cWord := C.CString(word)
	defer C.free(unsafe.Pointer(cWord))

	var resultPointer *C.varray

	code := C.varnam_transliterate(handle.connectionID, operationID, cWord, &resultPointer)

	if code == C.VARNAM_SUCCESS {
		resultChannel <- cgoVarnamTransliterateResult{
			resultPointer,
			nil,
		}
	} else {
		resultChannel <- cgoVarnamTransliterateResult{
			resultPointer,
			fmt.Errorf(handle.GetLastError()),
		}
	}

	close(resultChannel)
}

// Transliterate transilterate
func (handle *VarnamHandle) Transliterate(ctx context.Context, word string) ([]Suggestion, error) {
	var result []Suggestion

	operationID := makeContextOperation()
	channel := make(chan cgoVarnamTransliterateResult)

	go handle.cgoVarnamTransliterate(operationID, channel, word)

	select {
	case <-ctx.Done():
		C.varnam_cancel(operationID)
		return result, nil
	case channelResult := <-channel:
		if channelResult.err != nil {
			return result, channelResult.err
		}

		i := 0
		for i < int(C.varray_length(channelResult.result)) {
			cSug := (*C.Suggestion)(C.varray_get(channelResult.result, C.int(i)))
			sug := makeSuggestion(cSug)
			result = append(result, sug)
			i++
		}

		go C.destroySuggestionsArray(channelResult.result)

		return result, nil
	}
}

type cgoVarnamTransliterateAdvancedResult struct {
	result *C.struct_TransliterationResult_t
	err    error
}

func (handle *VarnamHandle) cgoVarnamTransliterateAdvanced(operationID C.int, resultChannel chan<- cgoVarnamTransliterateAdvancedResult, word string) {
	cWord := C.CString(word)
	defer C.free(unsafe.Pointer(cWord))

	var resultPointer *C.struct_TransliterationResult_t

	code := C.varnam_transliterate_advanced(handle.connectionID, operationID, cWord, &resultPointer)
	if code == C.VARNAM_SUCCESS {
		resultChannel <- cgoVarnamTransliterateAdvancedResult{
			resultPointer,
			nil,
		}
	} else {
		resultChannel <- cgoVarnamTransliterateAdvancedResult{
			resultPointer,
			fmt.Errorf(handle.GetLastError()),
		}
	}

	close(resultChannel)
}

// TransliterateAdvanced transilterate
func (handle *VarnamHandle) TransliterateAdvanced(ctx context.Context, word string) (TransliterationResult, error) {
	var result TransliterationResult

	operationID := makeContextOperation()
	channel := make(chan cgoVarnamTransliterateAdvancedResult)

	go handle.cgoVarnamTransliterateAdvanced(operationID, channel, word)

	select {
	case <-ctx.Done():
		C.varnam_cancel(operationID)
		return result, nil
	case channelResult := <-channel:
		if channelResult.err != nil {
			return result, channelResult.err
		}
		result = makeGoTransliterationResult(ctx, channelResult.result)
		return result, nil
	}
}

// TransliterateGreedyTokenized transliterate but only tokenizer output
func (handle *VarnamHandle) TransliterateGreedyTokenized(word string) []Suggestion {
	var result []Suggestion

	var resultPointer *C.varray

	cWord := C.CString(word)
	defer C.free(unsafe.Pointer(cWord))

	code := C.varnam_transliterate_greedy_tokenized(handle.connectionID, cWord, &resultPointer)
	if code != C.VARNAM_SUCCESS {
		log.Print(handle.GetLastError())
		return result
	}

	i := 0
	for i < int(C.varray_length(resultPointer)) {
		cSug := (*C.Suggestion)(C.varray_get(resultPointer, C.int(i)))
		sug := makeSuggestion(cSug)
		result = append(result, sug)
		i++
	}

	return result
}

// ReverseTransliterate reverse transilterate
func (handle *VarnamHandle) ReverseTransliterate(word string) ([]Suggestion, error) {
	var sugs []Suggestion

	cWord := C.CString(word)
	defer C.free(unsafe.Pointer(cWord))

	var resultPointer *C.varray
	defer C.destroySuggestionsArray(resultPointer)

	code := C.varnam_reverse_transliterate(handle.connectionID, cWord, &resultPointer)
	if code != C.VARNAM_SUCCESS {
		return sugs, &VarnamError{
			ErrorCode: int(code),
			Message:   handle.GetLastError(),
		}
	}

	i := 0
	for i < int(C.varray_length(resultPointer)) {
		cSug := (*C.Suggestion)(C.varray_get(resultPointer, C.int(i)))
		sug := makeSuggestion(cSug)
		sugs = append(sugs, sug)
		i++
	}
	return sugs, nil
}

// Train train a pattern => word
func (handle *VarnamHandle) Train(pattern string, word string) error {
	cPattern := C.CString(pattern)
	cWord := C.CString(word)

	err := C.varnam_train(handle.connectionID, cPattern, cWord)

	C.free(unsafe.Pointer(cPattern))
	C.free(unsafe.Pointer(cWord))

	return handle.checkError(err)
}

// Learn a word
func (handle *VarnamHandle) Learn(word string, weight int) error {
	cWord := C.CString(word)

	err := C.varnam_learn(handle.connectionID, cWord, C.int(weight))

	C.free(unsafe.Pointer(cWord))

	return handle.checkError(err)
}

// Unlearn a word
func (handle *VarnamHandle) Unlearn(word string) error {
	cWord := C.CString(word)

	err := C.varnam_unlearn(handle.connectionID, cWord)

	C.free(unsafe.Pointer(cWord))

	return handle.checkError(err)
}

// LearnFromFile learn words from a file
func (handle *VarnamHandle) LearnFromFile(filePath string) (LearnStatus, error) {
	var learnStatus LearnStatus

	cFilePath := C.CString(filePath)
	defer C.free(unsafe.Pointer(cFilePath))

	var resultPointer *C.LearnStatus
	defer C.free(unsafe.Pointer(resultPointer))

	code := C.varnam_learn_from_file(handle.connectionID, cFilePath, &resultPointer)
	if code != C.VARNAM_SUCCESS {
		return learnStatus, &VarnamError{
			ErrorCode: int(code),
			Message:   handle.GetLastError(),
		}
	}

	learnStatus = LearnStatus{
		int((*resultPointer).TotalWords),
		int((*resultPointer).FailedWords),
	}

	return learnStatus, nil
}

// TrainFromFile train pattern => word from a file
func (handle *VarnamHandle) TrainFromFile(filePath string) (LearnStatus, error) {
	var learnStatus LearnStatus

	cFilePath := C.CString(filePath)
	defer C.free(unsafe.Pointer(cFilePath))

	var resultPointer *C.LearnStatus
	defer C.free(unsafe.Pointer(resultPointer))

	code := C.varnam_train_from_file(handle.connectionID, cFilePath, &resultPointer)
	if code != C.VARNAM_SUCCESS {
		return learnStatus, &VarnamError{
			ErrorCode: int(code),
			Message:   handle.GetLastError(),
		}
	}

	learnStatus = LearnStatus{
		int((*resultPointer).TotalWords),
		int((*resultPointer).FailedWords),
	}

	return learnStatus, nil
}

// Export learnigns to a file
func (handle *VarnamHandle) Export(filePath string, wordsPerFile int) error {
	cFilePath := C.CString(filePath)
	err := C.varnam_export(handle.connectionID, cFilePath, C.int(wordsPerFile))
	return handle.checkError(err)
}

// Import learnigns to a file
func (handle *VarnamHandle) Import(filePath string) error {
	cFilePath := C.CString(filePath)
	err := C.varnam_import(handle.connectionID, cFilePath)
	return handle.checkError(err)
}

// GetRecentlyLearntWords get recently learn words
func (handle *VarnamHandle) GetRecentlyLearntWords(ctx context.Context, offset int, limit int) ([]Suggestion, error) {
	var result []Suggestion

	operationID := makeContextOperation()

	select {
	case <-ctx.Done():
		C.varnam_cancel(operationID)
		return result, nil
	default:
		var resultPointer *C.varray

		code := C.varnam_get_recently_learned_words(handle.connectionID, operationID, C.int(offset), C.int(limit), &resultPointer)
		if code != C.VARNAM_SUCCESS {
			return result, &VarnamError{
				ErrorCode: int(code),
				Message:   handle.GetLastError(),
			}
		}

		i := 0
		for i < int(C.varray_length(resultPointer)) {
			cSug := (*C.Suggestion)(C.varray_get(resultPointer, C.int(i)))
			sug := makeSuggestion(cSug)
			result = append(result, sug)
			i++
		}

		return result, nil
	}
}

// GetSuggestions get suggestions for a word
func (handle *VarnamHandle) GetSuggestions(ctx context.Context, word string) ([]Suggestion, error) {
	var result []Suggestion

	operationID := makeContextOperation()

	select {
	case <-ctx.Done():
		C.varnam_cancel(operationID)
		return result, nil
	default:
		var resultPointer *C.varray

		cWord := C.CString(word)
		defer C.free(unsafe.Pointer(cWord))

		code := C.varnam_get_suggestions(handle.connectionID, operationID, cWord, &resultPointer)
		if code != C.VARNAM_SUCCESS {
			return result, &VarnamError{
				ErrorCode: int(code),
				Message:   handle.GetLastError(),
			}
		}

		i := 0
		for i < int(C.varray_length(resultPointer)) {
			cSug := (*C.Suggestion)(C.varray_get(resultPointer, C.int(i)))
			sug := makeSuggestion(cSug)
			result = append(result, sug)
			i++
		}

		return result, nil
	}
}

func makeGoSchemeDetails(cSD *C.struct_SchemeDetails_t) SchemeDetails {
	isStable := true
	if cSD.IsStable == 0 {
		isStable = false
	}

	return SchemeDetails{
		C.GoString(cSD.Identifier),
		C.GoString(cSD.LangCode),
		C.GoString(cSD.DisplayName),
		C.GoString(cSD.Author),
		C.GoString(cSD.CompiledDate),
		isStable,
	}
}

// GetSchemeDetails get scheme details
func (handle *VarnamHandle) GetSchemeDetails() SchemeDetails {
	return makeGoSchemeDetails(C.varnam_get_scheme_details(handle.connectionID))
}

// GetVSTPath Get path to VST of current handle
func (handle *VarnamHandle) GetVSTPath() string {
	cStr := C.varnam_get_vst_path(handle.connectionID)
	defer C.free(unsafe.Pointer(cStr))
	return C.GoString(cStr)
}

func makeGoSymbol(cSymbol *C.Symbol) Symbol {
	var goSymbol Symbol
	goSymbol.Identifier = int(cSymbol.Identifier)
	goSymbol.Type = int(cSymbol.Type)
	goSymbol.MatchType = int(cSymbol.MatchType)
	goSymbol.Pattern = C.GoString(cSymbol.Pattern)
	goSymbol.Value1 = C.GoString(cSymbol.Value1)
	goSymbol.Value2 = C.GoString(cSymbol.Value2)
	goSymbol.Value3 = C.GoString(cSymbol.Value3)
	goSymbol.Tag = C.GoString(cSymbol.Tag)
	goSymbol.Weight = int(cSymbol.Weight)
	goSymbol.Priority = int(cSymbol.Priority)
	goSymbol.AcceptCondition = int(cSymbol.AcceptCondition)
	goSymbol.Flags = int(cSymbol.Flags)
	return goSymbol
}

func NewSearchSymbol() Symbol {
	var resultPointer *C.Symbol
	C.varnam_new_search_symbol(&resultPointer)
	return makeGoSymbol(resultPointer)
}

// SearchSymbolTable search VST
func (handle *VarnamHandle) SearchSymbolTable(ctx context.Context, searchCriteria Symbol) []Symbol {
	var goResults []Symbol

	operationID := makeContextOperation()

	select {
	case <-ctx.Done():
		return goResults
	default:
		Identifier := C.int(searchCriteria.Identifier)
		Type := C.int(searchCriteria.Type)
		MatchType := C.int(searchCriteria.MatchType)
		Pattern := C.CString(searchCriteria.Pattern)
		Value1 := C.CString(searchCriteria.Value1)
		Value2 := C.CString(searchCriteria.Value2)
		Value3 := C.CString(searchCriteria.Value3)
		Tag := C.CString(searchCriteria.Tag)
		Weight := C.int(searchCriteria.Weight)
		Priority := C.int(searchCriteria.Priority)
		AcceptCondition := C.int(searchCriteria.AcceptCondition)
		Flags := C.int(searchCriteria.Flags)

		symbol := C.makeSymbol(Identifier, Type, MatchType, Pattern, Value1, Value2, Value3, Tag, Weight, Priority, AcceptCondition, Flags)

		var resultPointer *C.varray
		defer C.destroySymbolArray(unsafe.Pointer(resultPointer))

		code := C.varnam_search_symbol_table(handle.connectionID, operationID, *symbol, &resultPointer)

		if code != C.VARNAM_SUCCESS {
			return goResults
		}

		i := 0
		for i < int(C.varray_length(resultPointer)) {
			result := (*C.Symbol)(C.varray_get(resultPointer, C.int(i)))

			goResults = append(goResults, makeGoSymbol(result))
			i++
		}

		return goResults
	}
}

// GetVSTDir Get path to directory containging the VSTs
func GetVSTDir() string {
	cStr := C.varnam_get_vst_dir()
	defer C.free(unsafe.Pointer(cStr))
	return C.GoString(cStr)
}

// GetAllSchemeDetails get all available scheme details. The bool is for error
func GetAllSchemeDetails() ([]SchemeDetails, bool) {
	cSchemeDetails := C.varnam_get_all_scheme_details()

	if cSchemeDetails == nil {
		return nil, true
	}

	var schemeDetails []SchemeDetails
	i := 0
	for i < int(C.varray_length(cSchemeDetails)) {
		cSD := (*C.SchemeDetails)(C.varray_get(cSchemeDetails, C.int(i)))
		schemeDetails = append(schemeDetails, makeGoSchemeDetails(cSD))
		i++
	}

	go C.destroySchemeDetailsArray(unsafe.Pointer(cSchemeDetails))

	return schemeDetails, false
}
