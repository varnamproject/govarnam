package govarnamgo

import "testing"

func TestReIndex(t *testing.T) {
	varnam := getVarnamInstance("ml")

	err := varnam.ReIndexDictionary()
	assertEqual(t, err, nil)
}
