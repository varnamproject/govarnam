package main

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

	"gitlab.com/subins2000/govarnam/govarnam"
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

func makeCTransliterationResult(ctx context.Context, goResult govarnam.TransliterationResult) *C.struct_TransliterationResult_t {
	select {
	case <-ctx.Done():
		return nil
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

		return C.makeResult(cExactMatch, cDictionarySuggestions, cPatternDictionarySuggestions, cTokenizerSuggestions, cGreedyTokenized)
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
func varnam_transliterate(varnamHandleID C.int, word *C.char) *C.struct_TransliterationResult_t {
	return makeCTransliterationResult(backgroundContext, getVarnamHandle(varnamHandleID).varnam.Transliterate(C.GoString(word)))
}

//export varnam_reverse_transliterate
func varnam_reverse_transliterate(varnamHandleID C.int, word *C.char) *C.varray {
	handle := getVarnamHandle(varnamHandleID)
	sugs, err := handle.varnam.ReverseTransliterate(C.GoString(word))

	if err != nil {
		handle.err = err
		return nil
	}

	cVArray := C.varray_init()
	for _, sug := range sugs {
		cSug := unsafe.Pointer(C.makeSuggestion(C.CString(sug.Word), C.int(sug.Weight), C.int(sug.LearnedOn)))
		C.varray_push(cVArray, cSug)
	}

	return cVArray
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
func varnam_learn_from_file(varnamHandleID C.int, filePath *C.char) C.int {
	handle := getVarnamHandle(varnamHandleID)
	handle.err = handle.varnam.LearnFromFile(C.GoString(filePath))
	return checkError(handle.err)
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

//export varnam_transliterate_with_id
func varnam_transliterate_with_id(varnamHandleID C.int, id C.int, word *C.char) *C.struct_TransliterationResult_t {
	ctx, cancel := context.WithCancel(backgroundContext)
	defer cancel()

	cancelFuncsMapMutex.Lock()
	cancelFuncs[id] = &cancel
	cancelFuncsMapMutex.Unlock()

	channel := make(chan govarnam.TransliterationResult)

	go getVarnamHandle(varnamHandleID).varnam.TransliterateWithContext(ctx, C.GoString(word), channel)

	select {
	case <-ctx.Done():
		return nil
	case result := <-channel:
		cResult := makeCTransliterationResult(ctx, result)
		return cResult
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
	} else {
		return C.VARNAM_ERROR
	}
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
