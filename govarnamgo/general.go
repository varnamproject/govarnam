package govarnamgo

// #cgo pkg-config: govarnam
// #include "libgovarnam.h"
// #include "stdlib.h"
import "C"

func (handle *VarnamHandle) ReIndexDictionary() error {
	err := C.varnam_reindex_dictionary(handle.connectionID)
	return handle.checkError(err)
}
