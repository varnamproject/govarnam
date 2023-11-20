package govarnamgo

import (
	"log"
	"os"
	"os/exec"
	"reflect"
	"runtime/debug"
	"strings"
	"sync"
	"testing"
)

var varnamInstances = map[string]*VarnamHandle{}
var mutex = sync.RWMutex{}
var testTempDir string

func checkError(err error) {
	if err != nil {
		log.Fatal(err.Error())
	}
}

// AssertEqual checks if values are equal
// Thanks https://gist.github.com/samalba/6059502#gistcomment-2710184
func assertEqual(t *testing.T, a interface{}, b interface{}) {
	if a == b {
		return
	}
	debug.PrintStack()
	t.Errorf("Received %v (type %v), expected %v (type %v)", a, reflect.TypeOf(a), b, reflect.TypeOf(b))
}

func setUp(schemeID string) {
	varnam, err := InitFromID(schemeID)
	checkError(err)

	mutex.Lock()
	varnamInstances[schemeID] = varnam
	mutex.Unlock()
}

func getVarnamInstance(schemeID string) *VarnamHandle {
	mutex.Lock()
	instance, ok := varnamInstances[schemeID]
	mutex.Unlock()

	if ok {
		return instance
	}
	return nil
}

func TestVersion(t *testing.T) {
	cmd := "echo $(git describe --abbrev=0 --tags || echo 'latest') | sed s/v//"
	cmdRun, buff := exec.Command("bash", "-c", cmd), new(strings.Builder)
	cmdRun.Stdout = buff
	cmdRun.Run()
	tagVersion := strings.TrimRight(buff.String(), "\n")

	assertEqual(t, GetVersion(), tagVersion)
}

func tearDown() {
	os.RemoveAll(testTempDir)
}

func TestMain(m *testing.M) {
	var err error
	testTempDir, err = os.MkdirTemp("", "govarnamgo_test")
	checkError(err)

	setUp("ml")
	setUp("ml-inscript")
	m.Run()
	tearDown()
}
