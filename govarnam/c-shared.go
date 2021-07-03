package govarnam

/*
#include "c-shared.h"
#include "c-shared-varray.h"
*/
import "C"
import (
	"context"
	"sync"
	"unsafe"
)

const okCode = 0
const errCode = 1

var varnamObj *Varnam
var mtx sync.Mutex

//export cInitFromLang
func cInitFromLang(langCode *C.char) int {
	mtx.Lock()
	defer mtx.Unlock()

	var err error
	varnamObj, err = InitFromLang(C.GoString(langCode))

	if err != nil {
		return errCode
	}
	return okCode
}

//export cTransliterate
func cTransliterate(word *C.char, callback *C.TransliterateCallbackFn) {
	goResult := varnamObj.Transliterate(C.GoString(word))

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

	C.callTransliterateCallback(*callback, cExactMatch, cSuggestions, cGreedyTokenized, C.int(goResult.DictionaryResultCount))
}

//export cTransliterateWithContext
func cTransliterateWithContext(ctx unsafe.Pointer, word *C.char) {
	varnamObj.TransliterateWithContext(*(*context.Context)(ctx), C.GoString(word))
}
