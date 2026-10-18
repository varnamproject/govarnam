//go:build linux && cgo
// +build linux,cgo

package govarnam

/*
#include <malloc.h>

// mallinfo2() is only available on glibc >= 2.33
static size_t cHeapUsed() {
#if defined(__GLIBC__) && (__GLIBC__ > 2 || (__GLIBC__ == 2 && __GLIBC_MINOR__ >= 33))
	return mallinfo2().uordblks;
#else
	struct mallinfo mi = mallinfo();
	return (size_t) mi.uordblks;
#endif
}
*/
import "C"

import (
	"os"
	"path/filepath"
	"testing"
)

// Prepared statements that are never closed keep their sqlite3_stmt allocation
// alive until the connection is closed, because database/sql keeps the driver
// statement referenced from the connection. These tests guard against that.
const maxCHeapGrowthPerCall = 1024 // bytes

func newLeakTestVarnam(t *testing.T) *Varnam {
	t.Helper()

	vstDir := os.Getenv("VARNAM_VST_DIR")
	if vstDir == "" {
		t.Skip("VARNAM_VST_DIR is not set")
	}

	dir := t.TempDir()
	varnam, err := Init(filepath.Join(vstDir, "ml.vst"), filepath.Join(dir, "learnings.vst"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		varnam.Close()
	})

	return varnam
}

func assertNoCHeapGrowth(t *testing.T, name string, calls int, fn func() error) {
	t.Helper()

	for i := 0; i < 200; i++ { // warm up
		if err := fn(); err != nil {
			t.Fatal(err)
		}
	}

	before := uint64(C.cHeapUsed())

	for i := 0; i < calls; i++ {
		if err := fn(); err != nil {
			t.Fatal(err)
		}
	}

	after := uint64(C.cHeapUsed())
	if after <= before {
		return
	}

	if grew := after - before; grew > uint64(calls*maxCHeapGrowthPerCall) {
		t.Errorf("%s: C heap grew by %d bytes over %d calls (%d B/call). sqlite3 statements are probably not being closed",
			name, grew, calls, grew/uint64(calls))
	}
}

func TestImportDoesNotLeakCHeap(t *testing.T) {
	varnam := newLeakTestVarnam(t)

	importPath := filepath.Join(t.TempDir(), "import.vlf")
	if err := os.WriteFile(importPath, []byte(`{"words":[{"w":"മലയാളം","c":1,"l":1}],"patterns":[]}`), 0644); err != nil {
		t.Fatal(err)
	}

	assertNoCHeapGrowth(t, "Import", 2000, func() error {
		return varnam.Import(importPath)
	})
}

func TestLearnManyDoesNotLeakCHeap(t *testing.T) {
	varnam := newLeakTestVarnam(t)

	words := []WordInfo{{word: "മലയാളം", weight: 1}}

	assertNoCHeapGrowth(t, "LearnMany", 2000, func() error {
		_, err := varnam.LearnMany(words)
		return err
	})
}
