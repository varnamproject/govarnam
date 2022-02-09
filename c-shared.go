package main

/**
 * govarnam - An Indian language transliteration library
 * Copyright Subin Siby <mail at subinsb (.) com>, 2021
 * Licensed under AGPL-3.0-only. See LICENSE.txt
 */

/* For c-shared library */

/*
#include "c-shared.h"
#include "c-shared-varray.h"
#include "stdlib.h"
*/
import "C"
import (
	"context"
	"log"
	"sync"
	"unsafe"

	"github.com/varnamproject/govarnam/govarnam"
)

var generalError error

var backgroundContext = context.Background()
var cancelFuncs = map[C.int]interface{}{}
var cancelFuncsMapMutex = sync.RWMutex{}

// Returns a C integer status code
// In the C world, errors are indicated by return int status codes
func checkError(err error) C.int {
	if err != nil {
		return C.VARNAM_ERROR
	}
	return C.VARNAM_SUCCESS
}

// In C, booleans are implemented with int 0 & int 1
func cintToBool(val C.int) bool {
	if val == C.int(1) {
		return true
	}
	return false
}

func makeContext(id C.int) (context.Context, func()) {
	ctx, cancel := context.WithCancel(backgroundContext)

	cancelFuncsMapMutex.Lock()
	cancelFuncs[id] = &cancel
	cancelFuncsMapMutex.Unlock()

	return ctx, cancel
}

func makeCTransliterationResult(ctx context.Context, goResult govarnam.TransliterationResult, resultPointer **C.struct_TransliterationResult_t) C.int {
	select {
	case <-ctx.Done():
		return C.VARNAM_CANCELLED
	default:
		// Note that C.CString uses malloc()
		// They should be freed manually. GC won't pick it.
		// The freeing should be done by programs using govarnam

		cExactWords := C.varray_init()
		for _, sug := range goResult.ExactWords {
			cSug := unsafe.Pointer(C.makeSuggestion(C.CString(sug.Word), C.int(sug.Weight), C.int(sug.LearnedOn)))
			C.varray_push(cExactWords, cSug)
		}

		cExactMatches := C.varray_init()
		for _, sug := range goResult.ExactMatches {
			cSug := unsafe.Pointer(C.makeSuggestion(C.CString(sug.Word), C.int(sug.Weight), C.int(sug.LearnedOn)))
			C.varray_push(cExactMatches, cSug)
		}

		cDictionarySuggestions := C.varray_init()
		for _, sug := range goResult.DictionarySuggestions {
			cSug := unsafe.Pointer(C.makeSuggestion(C.CString(sug.Word), C.int(sug.Weight), C.int(sug.LearnedOn)))
			C.varray_push(cDictionarySuggestions, cSug)
		}

		cPatternDictionarySuggestions := C.varray_init()
		for _, sug := range goResult.PatternDictionarySuggestions {
			cSug := unsafe.Pointer(C.makeSuggestion(C.CString(sug.Word), C.int(sug.Weight), C.int(sug.LearnedOn)))
			C.varray_push(cPatternDictionarySuggestions, cSug)
		}

		cTokenizerSuggestions := C.varray_init()
		for _, sug := range goResult.TokenizerSuggestions {
			cSug := unsafe.Pointer(C.makeSuggestion(C.CString(sug.Word), C.int(sug.Weight), C.int(sug.LearnedOn)))
			C.varray_push(cTokenizerSuggestions, cSug)
		}

		cGreedyTokenized := C.varray_init()
		for _, sug := range goResult.GreedyTokenized {
			cSug := unsafe.Pointer(C.makeSuggestion(C.CString(sug.Word), C.int(sug.Weight), C.int(sug.LearnedOn)))
			C.varray_push(cGreedyTokenized, cSug)
		}

		*resultPointer = C.makeResult(cExactWords, cExactMatches, cDictionarySuggestions, cPatternDictionarySuggestions, cTokenizerSuggestions, cGreedyTokenized)

		return C.VARNAM_SUCCESS
	}
}

//export varnam_get_version
func varnam_get_version() *C.char {
	return C.CString(govarnam.VersionString)
}

//export varnam_get_build
func varnam_get_build() *C.char {
	return C.CString(govarnam.BuildString)
}

//export varnam_set_vst_lookup_dir
func varnam_set_vst_lookup_dir(path *C.char) {
	govarnam.SetVSTLookupDir(C.GoString(path))
}

type varnamHandle struct {
	varnam *govarnam.Varnam
	err    error
}

// For storing varnam instances
var varnamHandles = map[C.int]*varnamHandle{}
var varnamHandlesMapMutex = sync.RWMutex{}

//export varnam_init
func varnam_init(vstFile *C.char, learningsFile *C.char, id unsafe.Pointer) C.int {
	handleID := C.int(len(varnamHandles))
	*(*C.int)(id) = handleID

	varnamGo, err := govarnam.Init(C.GoString(vstFile), C.GoString(learningsFile))

	varnamHandlesMapMutex.Lock()
	varnamHandles[handleID] = &varnamHandle{varnamGo, err}
	varnamHandlesMapMutex.Unlock()

	return checkError(err)
}

//export varnam_init_from_id
func varnam_init_from_id(schemeID *C.char, id unsafe.Pointer) C.int {
	handleID := C.int(len(varnamHandles))
	*(*C.int)(id) = handleID

	varnamGo, err := govarnam.InitFromID(C.GoString(schemeID))

	varnamHandlesMapMutex.Lock()
	varnamHandles[handleID] = &varnamHandle{varnamGo, err}
	varnamHandlesMapMutex.Unlock()

	return checkError(err)
}

func getVarnamHandle(id C.int) *varnamHandle {
	varnamHandlesMapMutex.Lock()
	defer varnamHandlesMapMutex.Unlock()
	if handle, ok := varnamHandles[id]; ok {
		return handle
	}
	log.Fatal("Varnam handle not found")
	return &varnamHandle{}
}

//export varnam_close
func varnam_close(varnamHandleID C.int) C.int {
	handle := getVarnamHandle(varnamHandleID)
	handle.err = handle.varnam.Close()

	if handle.err != nil {
		return checkError(handle.err)
	}

	varnamHandlesMapMutex.Lock()
	delete(varnamHandles, varnamHandleID)
	varnamHandlesMapMutex.Unlock()

	return C.VARNAM_SUCCESS
}

//export varnam_transliterate
func varnam_transliterate(varnamHandleID C.int, id C.int, word *C.char, resultPointer **C.varray) C.int {
	ctx, cancel := makeContext(id)
	defer cancel()

	channel := make(chan []govarnam.Suggestion)

	go getVarnamHandle(varnamHandleID).varnam.TransliterateWithContext(ctx, C.GoString(word), channel)

	select {
	case <-ctx.Done():
		return C.VARNAM_CANCELLED
	case result := <-channel:
		// Note that C.CString uses malloc()
		// They should be freed manually. GC won't pick it.
		// The freeing should be done by programs using govarnam

		cResult := C.varray_init()
		for _, sug := range result {
			cSug := unsafe.Pointer(C.makeSuggestion(C.CString(sug.Word), C.int(sug.Weight), C.int(sug.LearnedOn)))
			C.varray_push(cResult, cSug)
		}
		*resultPointer = cResult

		return C.VARNAM_SUCCESS
	}
}

//export varnam_transliterate_advanced
func varnam_transliterate_advanced(varnamHandleID C.int, id C.int, word *C.char, resultPointer **C.struct_TransliterationResult_t) C.int {
	ctx, cancel := makeContext(id)
	defer cancel()

	channel := make(chan govarnam.TransliterationResult)

	go getVarnamHandle(varnamHandleID).varnam.TransliterateAdvancedWithContext(ctx, C.GoString(word), channel)

	select {
	case <-ctx.Done():
		return C.VARNAM_CANCELLED
	case result := <-channel:
		return makeCTransliterationResult(ctx, result, resultPointer)
	}
}

//export varnam_transliterate_greedy_tokenized
func varnam_transliterate_greedy_tokenized(varnamHandleID C.int, word *C.char, resultPointer **C.varray) C.int {
	handle := getVarnamHandle(varnamHandleID)

	result := handle.varnam.TransliterateGreedyTokenized(C.GoString(word))

	ptr := C.varray_init()
	for _, sug := range result {
		cSug := unsafe.Pointer(C.makeSuggestion(C.CString(sug.Word), C.int(sug.Weight), C.int(sug.LearnedOn)))
		C.varray_push(ptr, cSug)
	}
	*resultPointer = ptr

	return C.VARNAM_SUCCESS
}

//export varnam_reverse_transliterate
func varnam_reverse_transliterate(varnamHandleID C.int, word *C.char, resultPointer **C.varray) C.int {
	handle := getVarnamHandle(varnamHandleID)
	sugs, err := handle.varnam.ReverseTransliterate(C.GoString(word))

	if err != nil {
		handle.err = err
		return C.VARNAM_ERROR
	}

	cResult := C.varray_init()
	for _, sug := range sugs {
		cSug := unsafe.Pointer(C.makeSuggestion(C.CString(sug.Word), C.int(sug.Weight), C.int(sug.LearnedOn)))
		C.varray_push(cResult, cSug)
	}
	*resultPointer = cResult

	return C.VARNAM_SUCCESS
}

//export varnam_debug
func varnam_debug(varnamHandleID C.int, val C.int) {
	getVarnamHandle(varnamHandleID).varnam.Debug = cintToBool(val)
}

// Deprecated. Use varnam_config()
//export varnam_set_indic_digits
func varnam_set_indic_digits(varnamHandleID C.int, val C.int) {
	varnam_config(varnamHandleID, C.VARNAM_CONFIG_USE_INDIC_DIGITS, val)
}

// Deprecated. Use varnam_config()
//export varnam_set_dictionary_suggestions_limit
func varnam_set_dictionary_suggestions_limit(varnamHandleID C.int, val C.int) {
	getVarnamHandle(varnamHandleID).varnam.DictionarySuggestionsLimit = int(val)
}

// Deprecated. Use varnam_config()
//export varnam_set_pattern_dictionary_suggestions_limit
func varnam_set_pattern_dictionary_suggestions_limit(varnamHandleID C.int, val C.int) {
	getVarnamHandle(varnamHandleID).varnam.PatternDictionarySuggestionsLimit = int(val)
}

// Deprecated. Use varnam_config()
//export varnam_set_tokenizer_suggestions_limit
func varnam_set_tokenizer_suggestions_limit(varnamHandleID C.int, val C.int) {
	getVarnamHandle(varnamHandleID).varnam.TokenizerSuggestionsLimit = int(val)
}

// Deprecated. Use varnam_config()
//export varnam_set_dictionary_match_exact
func varnam_set_dictionary_match_exact(varnamHandleID C.int, val C.int) {
	if val == 0 {
		getVarnamHandle(varnamHandleID).varnam.DictionaryMatchExact = false
	} else {
		getVarnamHandle(varnamHandleID).varnam.DictionaryMatchExact = true
	}
}

//export varnam_learn
func varnam_learn(varnamHandleID C.int, word *C.char, weight C.int) C.int {
	handle := getVarnamHandle(varnamHandleID)
	handle.err = handle.varnam.Learn(C.GoString(word), int(weight))
	return checkError(handle.err)
}

//export varnam_train
func varnam_train(varnamHandleID C.int, pattern *C.char, word *C.char) C.int {
	handle := getVarnamHandle(varnamHandleID)
	handle.err = handle.varnam.Train(C.GoString(pattern), C.GoString(word))
	return checkError(handle.err)
}

//export varnam_unlearn
func varnam_unlearn(varnamHandleID C.int, word *C.char) C.int {
	handle := getVarnamHandle(varnamHandleID)
	handle.err = handle.varnam.Unlearn(C.GoString(word))
	return checkError(handle.err)
}

//export varnam_learn_from_file
func varnam_learn_from_file(varnamHandleID C.int, filePath *C.char, resultPointer **C.struct_LearnStatus_t) C.int {
	handle := getVarnamHandle(varnamHandleID)
	learnStatus, err := handle.varnam.LearnFromFile(C.GoString(filePath))

	if err != nil {
		handle.err = err
		return C.VARNAM_ERROR
	}

	result := C.makeLearnStatus(C.int(learnStatus.TotalWords), C.int(learnStatus.FailedWords))
	*resultPointer = &result

	return C.VARNAM_SUCCESS
}

//export varnam_train_from_file
func varnam_train_from_file(varnamHandleID C.int, filePath *C.char, resultPointer **C.struct_LearnStatus_t) C.int {
	handle := getVarnamHandle(varnamHandleID)

	learnStatus, err := handle.varnam.TrainFromFile(C.GoString(filePath))

	if err != nil {
		handle.err = err
		return C.VARNAM_ERROR
	}

	result := C.makeLearnStatus(C.int(learnStatus.TotalWords), C.int(learnStatus.FailedWords))
	*resultPointer = &result

	return C.VARNAM_SUCCESS
}

//export varnam_get_last_error
func varnam_get_last_error(varnamHandleID C.int) *C.char {
	var err error

	if varnamHandleID == -1 {
		err = generalError
	} else {
		err = getVarnamHandle(varnamHandleID).err
	}

	if err != nil {
		return C.CString(err.Error())
	} else {
		return C.CString("")
	}
}

//export varnam_cancel
func varnam_cancel(id C.int) C.int {
	cancelFuncsMapMutex.Lock()
	cancelFunc, ok := cancelFuncs[id]
	defer cancelFuncsMapMutex.Unlock()

	if ok {
		(*cancelFunc.(*context.CancelFunc))()
		delete(cancelFuncs, id)
		return C.VARNAM_SUCCESS
	}
	return C.VARNAM_ERROR
}

//export varnam_export
func varnam_export(varnamHandleID C.int, filePath *C.char, wordsPerFile C.int) C.int {
	handle := getVarnamHandle(varnamHandleID)
	handle.err = handle.varnam.Export(C.GoString(filePath), int(wordsPerFile))

	return checkError(handle.err)
}

//export varnam_import
func varnam_import(varnamHandleID C.int, filePath *C.char) C.int {
	handle := getVarnamHandle(varnamHandleID)
	handle.err = handle.varnam.Import(C.GoString(filePath))

	return checkError(handle.err)
}

//export varnam_get_vst_path
func varnam_get_vst_path(varnamHandleID C.int) *C.char {
	handle := getVarnamHandle(varnamHandleID)

	return C.CString(handle.varnam.VSTPath)
}

//export varnam_new_search_symbol
func varnam_new_search_symbol(resultPointer **C.struct_Symbol_t) C.int {
	symbol := govarnam.NewSearchSymbol()
	*resultPointer = C.makeSymbol(
		C.int(symbol.Identifier),
		C.int(symbol.Type),
		C.int(symbol.MatchType),
		C.CString(symbol.Pattern),
		C.CString(symbol.Value1),
		C.CString(symbol.Value2),
		C.CString(symbol.Value3),
		C.CString(symbol.Tag),
		C.int(symbol.Weight),
		C.int(symbol.Priority),
		C.int(symbol.AcceptCondition),
		C.int(symbol.Flags),
	)

	return C.VARNAM_SUCCESS
}

//export varnam_search_symbol_table
func varnam_search_symbol_table(varnamHandleID C.int, id C.int, searchCriteria C.struct_Symbol_t, resultPointer **C.varray) C.int {
	ctx, cancel := makeContext(id)
	defer cancel()

	handle := getVarnamHandle(varnamHandleID)

	goSearchCriteria := cSymbolToGoSymbol(searchCriteria)

	var results []govarnam.Symbol

	select {
	case <-ctx.Done():
		return C.VARNAM_CANCELLED
	default:
		results, handle.err = handle.varnam.SearchSymbolTable(ctx, goSearchCriteria)

		cResult := C.varray_init()
		for _, symbol := range results {
			cSymbol := unsafe.Pointer(goSymbolToCSymbol(symbol))
			C.varray_push(cResult, cSymbol)
		}
		*resultPointer = cResult

		return C.VARNAM_SUCCESS
	}
}

//export varnam_get_recently_learned_words
func varnam_get_recently_learned_words(varnamHandleID C.int, id C.int, offset C.int, limit C.int, resultPointer **C.varray) C.int {
	ctx, cancel := makeContext(id)
	defer cancel()

	handle := getVarnamHandle(varnamHandleID)

	result, err := handle.varnam.GetRecentlyLearntWords(ctx, int(offset), int(limit))

	if err != nil {
		handle.err = err
		return C.VARNAM_ERROR
	}

	ptr := C.varray_init()
	for _, sug := range result {
		cSug := unsafe.Pointer(C.makeSuggestion(C.CString(sug.Word), C.int(sug.Weight), C.int(sug.LearnedOn)))
		C.varray_push(ptr, cSug)
	}
	*resultPointer = ptr

	return C.VARNAM_SUCCESS
}

//export varnam_get_suggestions
func varnam_get_suggestions(varnamHandleID C.int, id C.int, word *C.char, resultPointer **C.varray) C.int {
	ctx, cancel := makeContext(id)
	defer cancel()

	handle := getVarnamHandle(varnamHandleID)

	result := handle.varnam.GetSuggestions(ctx, C.GoString(word))

	ptr := C.varray_init()
	for _, sug := range result {
		cSug := unsafe.Pointer(C.makeSuggestion(C.CString(sug.Word), C.int(sug.Weight), C.int(sug.LearnedOn)))
		C.varray_push(ptr, cSug)
	}
	*resultPointer = ptr

	return C.VARNAM_SUCCESS
}

func makeGoSchemeDetails(sd *C.struct_SchemeDetails_t) govarnam.SchemeDetails {
	return govarnam.SchemeDetails{
		Identifier:   C.GoString(sd.Identifier),
		LangCode:     C.GoString(sd.LangCode),
		DisplayName:  C.GoString(sd.DisplayName),
		Author:       C.GoString(sd.Author),
		CompiledDate: C.GoString(sd.CompiledDate),
		IsStable:     cintToBool(sd.IsStable),
	}
}

func makeCSchemeDetails(sd govarnam.SchemeDetails) *C.struct_SchemeDetails_t {
	var cIsStable C.int

	if sd.IsStable {
		cIsStable = C.int(1)
	} else {
		cIsStable = C.int(0)
	}

	return C.makeSchemeDetails(
		C.CString(sd.Identifier),
		C.CString(sd.LangCode),
		C.CString(sd.DisplayName),
		C.CString(sd.Author),
		C.CString(sd.CompiledDate),
		cIsStable,
	)
}

//export varnam_get_scheme_details
func varnam_get_scheme_details(varnamHandleID C.int) *C.struct_SchemeDetails_t {
	handle := getVarnamHandle(varnamHandleID)
	return makeCSchemeDetails(handle.varnam.SchemeDetails)
}

//export varnam_get_vst_dir
func varnam_get_vst_dir() *C.char {
	var dir string
	dir, generalError = govarnam.FindVSTDir()
	return C.CString(dir)
}

//export varnam_get_all_scheme_details
func varnam_get_all_scheme_details() *C.varray {
	var schemeDetails []govarnam.SchemeDetails
	schemeDetails, generalError = govarnam.GetAllSchemeDetails()

	if generalError != nil {
		return nil
	}

	cSchemeDetails := C.varray_init()
	for _, sd := range schemeDetails {
		cSD := unsafe.Pointer(makeCSchemeDetails(sd))
		C.varray_push(cSchemeDetails, cSD)
	}

	return cSchemeDetails
}

//export vm_init
func vm_init(vstPath *C.char, id unsafe.Pointer) C.int {
	handleID := C.int(len(varnamHandles))
	*(*C.int)(id) = handleID

	varnamGo, err := govarnam.VMInit(C.GoString(vstPath))

	varnamHandlesMapMutex.Lock()
	varnamHandles[handleID] = &varnamHandle{varnamGo, err}
	varnamHandlesMapMutex.Unlock()

	return checkError(err)
}

//export vm_create_token
func vm_create_token(varnamHandleID C.int, pattern *C.char, value1 *C.char, value2 *C.char, value3 *C.char, tag *C.char, symbolType C.int, matchType C.int, priority C.int, acceptCondition C.int, buffered C.int) C.int {
	handle := getVarnamHandle(varnamHandleID)

	// if pattern == nil {
	// 	pattern = C.CString("")
	// }
	// if value1 == nil {
	// 	value1 = C.CString("")
	// }
	// if value2 == nil {
	// 	value2 = C.CString("")
	// }
	// if value3 == nil {
	// 	value3 = C.CString("")
	// }
	// if tag == nil {
	// 	tag = C.CString("")
	// }

	handle.err = handle.varnam.VMCreateToken(
		C.GoString(pattern),
		C.GoString(value1),
		C.GoString(value2),
		C.GoString(value3),
		C.GoString(tag),
		int(symbolType),
		int(matchType),
		int(priority),
		int(acceptCondition),
		cintToBool(buffered),
	)

	return checkError(handle.err)
}

//export vm_delete_token
func vm_delete_token(varnamHandleID C.int, searchCriteria C.struct_Symbol_t) C.int {
	handle := getVarnamHandle(varnamHandleID)

	goSearchCriteria := cSymbolToGoSymbol(searchCriteria)

	handle.err = handle.varnam.VMDeleteToken(goSearchCriteria)
	return checkError(handle.err)
}

//export vm_flush_buffer
func vm_flush_buffer(varnamHandleID C.int) C.int {
	handle := getVarnamHandle(varnamHandleID)

	handle.err = handle.varnam.VMFlushBuffer()
	return checkError(handle.err)
}

//export varnam_config
func varnam_config(varnamHandleID C.int, key C.int, value C.int) C.int {
	handle := getVarnamHandle(varnamHandleID)

	switch key {
	case C.VARNAM_CONFIG_USE_INDIC_DIGITS:
		handle.varnam.LangRules.IndicDigits = cintToBool(value)
		break
	case C.VARNAM_CONFIG_USE_DEAD_CONSONANTS:
		handle.varnam.VSTMakerConfig.UseDeadConsonants = cintToBool(value)
		break
	case C.VARNAM_CONFIG_IGNORE_DUPLICATE_TOKEN:
		handle.varnam.VSTMakerConfig.IgnoreDuplicateTokens = cintToBool(value)
		break
	case C.VARNAM_CONFIG_SET_DICTIONARY_SUGGESTIONS_LIMIT:
		handle.varnam.DictionarySuggestionsLimit = int(value)
		break
	case C.VARNAM_CONFIG_SET_PATTERN_DICTIONARY_SUGGESTIONS_LIMIT:
		handle.varnam.PatternDictionarySuggestionsLimit = int(value)
		break
	case C.VARNAM_CONFIG_SET_TOKENIZER_SUGGESTIONS_LIMIT:
		handle.varnam.TokenizerSuggestionsLimit = int(value)
		break
	case C.VARNAM_CONFIG_SET_DICTIONARY_MATCH_EXACT:
		handle.varnam.DictionaryMatchExact = cintToBool(value)
		break
	}

	return C.VARNAM_SUCCESS
}

//export vm_set_scheme_details
func vm_set_scheme_details(varnamHandleID C.int, sd *C.struct_SchemeDetails_t) C.int {
	handle := getVarnamHandle(varnamHandleID)

	handle.err = handle.varnam.VMSetSchemeDetails(makeGoSchemeDetails(sd))
	return checkError(handle.err)
}

func main() {}
