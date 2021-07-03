package main

/* For c-shared library */

/*
#include "c-shared.h"
#include "c-shared-varray.h"
*/
import "C"
import (
	"unsafe"

	"gitlab.com/subins2000/govarnam/govarnam"
)

func checkError(err error) C.int {
	if err != nil {
		return C.VARNAM_ERROR
	}
	return C.VARNAM_SUCCESS
}

// TODO limitation of using one varnam handle per library load.
// If Go pointers can be passed to C, multiple instances can be run
// But according to cgo, this isn't possible.
// Possible solution : Keep a hashmap on both C & Go with key as langCode
var varnamGo *govarnam.Varnam
var err error

//export varnam_init_from_id
func varnam_init_from_id(langCode *C.char) C.int {
	varnamGo, err = govarnam.InitFromLang(C.GoString(langCode))
	return checkError(err)
}

//export varnam_transliterate
func varnam_transliterate(word *C.char) *C.struct_TransliterationResult_t {
	goResult := varnamGo.Transliterate(C.GoString(word))

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

//export varnam_debug
func varnam_debug(val C.int) {
	if val == 0 {
		varnamGo.Debug = false
	} else {
		varnamGo.Debug = true
	}
}

//export varnam_set_indic_digits
func varnam_set_indic_digits(val C.int) {
	if val == 0 {
		varnamGo.LangRules.IndicDigits = false
	} else {
		varnamGo.LangRules.IndicDigits = true
	}
}

//export varnam_learn
func varnam_learn(word *C.char, confidence int) C.int {
	err = varnamGo.Learn(C.GoString(word), confidence)
	return checkError(err)
}

//export varnam_train
func varnam_train(pattern *C.char, word *C.char) C.int {
	err = varnamGo.Train(C.GoString(pattern), C.GoString(word))
	return checkError(err)
}

//export varnam_unlearn
func varnam_unlearn(word *C.char) C.int {
	err = varnamGo.Unlearn(C.GoString(word))
	return checkError(err)
}

//export varnam_get_last_error
func varnam_get_last_error() *C.char {
	return C.CString(err.Error())
}

func main() {}
