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
		for _, sug := range goResult.ExactMatch {
			cSug := unsafe.Pointer(C.makeSuggestion(C.CString(sug.Word), C.int(sug.Weight), C.int(sug.LearnedOn)))
			C.varray_push(cExactMatch, cSug)
		}

		cSuggestions := C.varray_init()
		for _, sug := range goResult.Suggestions {
			cSug := unsafe.Pointer(C.makeSuggestion(C.CString(sug.Word), C.int(sug.Weight), C.int(sug.LearnedOn)))
			C.varray_push(cSuggestions, cSug)
		}

		cGreedyTokenized := C.varray_init()
		for _, sug := range goResult.GreedyTokenized {
			cSug := unsafe.Pointer(C.makeSuggestion(C.CString(sug.Word), C.int(sug.Weight), C.int(sug.LearnedOn)))
			C.varray_push(cGreedyTokenized, cSug)
		}

		return C.makeResult(cExactMatch, cSuggestions, cGreedyTokenized, C.int(goResult.DictionaryResultCount))
	}
}

type varnamHandle struct {
	varnam *govarnam.Varnam
	err    error
}

// For storing varnam instances
var varnamHandles = map[C.int]interface{}{}

//export varnam_init_from_id
func varnam_init_from_id(langCode *C.char, id unsafe.Pointer) C.int {
	handleID := C.int(len(varnamHandles))
	*(*C.int)(id) = handleID

	varnamGo, err := govarnam.InitFromID(C.GoString(langCode))
	varnamHandles[handleID] = varnamHandle{varnamGo, err}

	return checkError(err)
}

func getVarnamHandle(id C.int) varnamHandle {
	if handle, ok := varnamHandles[id]; ok {
		return handle.(varnamHandle)
	} else {
		log.Fatal("Varnam handle not found")
		return varnamHandle{}
	}
}

//export varnam_transliterate
func varnam_transliterate(varnamHandleID C.int, word *C.char) *C.struct_TransliterationResult_t {
	return makeCTransliterationResult(backgroundContext, getVarnamHandle(varnamHandleID).varnam.Transliterate(C.GoString(word)))
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
func varnam_learn(varnamHandleID C.int, word *C.char, confidence C.int) C.int {
	handle := getVarnamHandle(varnamHandleID)
	handle.err = handle.varnam.Learn(C.GoString(word), int(confidence))
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

//export varnam_get_last_error
func varnam_get_last_error(varnamHandleID C.int) *C.char {
	return C.CString(getVarnamHandle(varnamHandleID).err.Error())
}

//export varnam_transliterate_with_id
func varnam_transliterate_with_id(varnamHandleID C.int, id C.int, word *C.char) *C.struct_TransliterationResult_t {
	ctx, cancel := context.WithCancel(backgroundContext)

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

func main() {}
