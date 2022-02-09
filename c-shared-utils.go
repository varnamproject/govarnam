package main

/*
#include "c-shared.h"
*/
import "C"

import "github.com/varnamproject/govarnam/govarnam"

func cSymbolToGoSymbol(symbol C.struct_Symbol_t) govarnam.Symbol {
	var goSymbol govarnam.Symbol
	goSymbol.Identifier = int(symbol.Identifier)
	goSymbol.Type = int(symbol.Type)
	goSymbol.MatchType = int(symbol.MatchType)
	goSymbol.Pattern = C.GoString(symbol.Pattern)
	goSymbol.Value1 = C.GoString(symbol.Value1)
	goSymbol.Value2 = C.GoString(symbol.Value2)
	goSymbol.Value3 = C.GoString(symbol.Value3)
	goSymbol.Tag = C.GoString(symbol.Tag)
	goSymbol.Weight = int(symbol.Weight)
	goSymbol.Priority = int(symbol.Priority)
	goSymbol.AcceptCondition = int(symbol.AcceptCondition)
	goSymbol.Flags = int(symbol.Flags)
	return goSymbol
}

func goSymbolToCSymbol(symbol govarnam.Symbol) *C.struct_Symbol_t {
	return C.makeSymbol(
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
}
