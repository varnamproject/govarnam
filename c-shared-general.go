package main

/*
#include "c-shared.h"
#include "c-shared-varray.h"
#include "stdlib.h"
*/
import "C"

//export varnam_reindex_dictionary
func varnam_reindex_dictionary(varnamHandleID C.int) C.int {
	handle := getVarnamHandle(varnamHandleID)
	handle.err = handle.varnam.ReIndexDictionary()

	return checkError(handle.err)
}
