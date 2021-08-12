package govarnamgo

import (
	"reflect"
	"runtime/debug"
	"testing"
)

// AssertEqual checks if values are equal
// Thanks https://gist.github.com/samalba/6059502#gistcomment-2710184
func assertEqual(t *testing.T, a interface{}, b interface{}) {
	if a == b {
		return
	}
	debug.PrintStack()
	t.Errorf("Received %v (type %v), expected %v (type %v)", a, reflect.TypeOf(a), b, reflect.TypeOf(b))
}

func TestSearchSymbolTable(t *testing.T) {
	varnam, _ := InitFromID("ml")

	var symbol Symbol
	symbol.Pattern = "la"
	result := varnam.SearchSymbolTable(symbol)

	assertEqual(t, result[0].Value1, "ല")
}

func TestMain(m *testing.M) {
	m.Run()
}
