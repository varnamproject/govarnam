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

func checkError(err error) C.int {
	if err != nil {
		return C.VARNAM_ERROR
	}
	return C.VARNAM_SUCCESS
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

		cExactMatch := C.varray_init()
		for _, sug := range goResult.ExactMatches {
			cSug := unsafe.Pointer(C.makeSuggestion(C.CString(sug.Word), C.int(sug.Weight), C.int(sug.LearnedOn)))
			C.varray_push(cExactMatch, cSug)
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

		*resultPointer = C.makeResult(cExactMatch, cDictionarySuggestions, cPatternDictionarySuggestions, cTokenizerSuggestions, cGreedyTokenized)

		return C.VARNAM_SUCCESS
	}
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

	channel := make(chan govarnam.TransliterationResult)

	go getVarnamHandle(varnamHandleID).varnam.TransliterateWithContext(ctx, C.GoString(word), channel)

	select {
	case <-ctx.Done():
		return C.VARNAM_CANCELLED
	case result := <-channel:
		// Note that C.CString uses malloc()
		// They should be freed manually. GC won't pick it.
		// The freeing should be done by programs using govarnam

		combined := result.ExactMatches
		combined = append(combined, result.PatternDictionarySuggestions...)
		combined = append(combined, result.DictionarySuggestions...)
		combined = append(combined, result.TokenizerSuggestions...)
		combined = append(combined, result.GreedyTokenized...)

		cResult := C.varray_init()
		for _, sug := range combined {
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

	go getVarnamHandle(varnamHandleID).varnam.TransliterateWithContext(ctx, C.GoString(word), channel)

	select {
	case <-ctx.Done():
		return C.VARNAM_CANCELLED
	case result := <-channel:
		return makeCTransliterationResult(ctx, result, resultPointer)
	}
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
	if val == 0 {
		getVarnamHandle(varnamHandleID).varnam.Debug = false
	} else {
		getVarnamHandle(varnamHandleID).varnam.Debug = true
	}
}

//export varnam_set_indic_digits
func varnam_set_indic_digits(varnamHandleID C.int, val C.int) {
	if val == 0 {
		getVarnamHandle(varnamHandleID).varnam.LangRules.IndicDigits = false
	} else {
		getVarnamHandle(varnamHandleID).varnam.LangRules.IndicDigits = true
	}
}

//export varnam_set_dictionary_suggestions_limit
func varnam_set_dictionary_suggestions_limit(varnamHandleID C.int, val C.int) {
	getVarnamHandle(varnamHandleID).varnam.DictionarySuggestionsLimit = int(val)
}

//export varnam_set_tokenizer_suggestions_limit
func varnam_set_tokenizer_suggestions_limit(varnamHandleID C.int, val C.int) {
	getVarnamHandle(varnamHandleID).varnam.TokenizerSuggestionsLimit = int(val)
}

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
func varnam_train_from_file(varnamHandleID C.int, filePath *C.char) C.int {
	handle := getVarnamHandle(varnamHandleID)
	handle.err = handle.varnam.TrainFromFile(C.GoString(filePath))
	return checkError(handle.err)
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
func varnam_export(varnamHandleID C.int, filePath *C.char) C.int {
	handle := getVarnamHandle(varnamHandleID)
	handle.err = handle.varnam.Export(C.GoString(filePath))

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

//export varnam_search_symbol_table
func varnam_search_symbol_table(varnamHandleID C.int, id C.int, searchCriteria C.struct_Symbol_t, resultPointer **C.varray) C.int {
	ctx, cancel := makeContext(id)
	defer cancel()

	handle := getVarnamHandle(varnamHandleID)

	var goSearchCriteria govarnam.Symbol
	goSearchCriteria.Identifier = int(searchCriteria.Identifier)
	goSearchCriteria.Type = int(searchCriteria.Type)
	goSearchCriteria.MatchType = int(searchCriteria.MatchType)
	goSearchCriteria.Pattern = C.GoString(searchCriteria.Pattern)
	goSearchCriteria.Value1 = C.GoString(searchCriteria.Value1)
	goSearchCriteria.Value2 = C.GoString(searchCriteria.Value2)
	goSearchCriteria.Value3 = C.GoString(searchCriteria.Value3)
	goSearchCriteria.Tag = C.GoString(searchCriteria.Tag)
	goSearchCriteria.Weight = int(searchCriteria.Weight)
	goSearchCriteria.Priority = int(searchCriteria.Priority)
	goSearchCriteria.AcceptCondition = int(searchCriteria.AcceptCondition)
	goSearchCriteria.Flags = int(searchCriteria.Flags)

	var results []govarnam.Symbol

	select {
	case <-ctx.Done():
		return C.VARNAM_CANCELLED
	default:
		results, handle.err = handle.varnam.SearchSymbolTable(ctx, goSearchCriteria)

		cResult := C.varray_init()
		for _, symbol := range results {
			cSymbol := unsafe.Pointer(C.makeSymbol(
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
			))
			C.varray_push(cResult, cSymbol)
		}
		*resultPointer = cResult

		return C.VARNAM_SUCCESS
	}
}

//export varnam_get_recently_learned_words
func varnam_get_recently_learned_words(varnamHandleID C.int, id C.int, limit C.int, resultPointer **C.varray) C.int {
	ctx, cancel := makeContext(id)
	defer cancel()

	handle := getVarnamHandle(varnamHandleID)

	result, err := handle.varnam.GetRecentlyLearntWords(ctx, int(limit))

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
		var cIsStable C.int

		if sd.IsStable {
			cIsStable = C.int(1)
		} else {
			cIsStable = C.int(0)
		}

		cSD := unsafe.Pointer(C.makeSchemeDetails(
			C.CString(sd.Identifier),
			C.CString(sd.LangCode),
			C.CString(sd.DisplayName),
			C.CString(sd.Author),
			C.CString(sd.CompiledDate),
			cIsStable,
		))
		C.varray_push(cSchemeDetails, cSD)
	}

	return cSchemeDetails
}

func main() {}
