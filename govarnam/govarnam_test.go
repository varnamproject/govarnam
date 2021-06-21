package govarnam

import (
	"io/ioutil"
	"log"
	"os"
	"path"
	"reflect"
	"runtime"
	"runtime/debug"
	"testing"
)

var (
	dictDir string
	varnam  Varnam
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

func setUp(langCode string) {
	_, filename, _, _ := runtime.Caller(0)
	projectRoot := path.Join(path.Dir(filename), "..")

	vstLoc := path.Join(projectRoot, "schemes", langCode+".vst")

	dictDir, err := ioutil.TempDir("", "govarnam_test")
	if err != nil {
		log.Fatal(err)
	}

	dictLoc := path.Join(dictDir, langCode+".vst.learnings")
	makeDictionary(dictLoc)

	varnam = Init(vstLoc, dictLoc)
}

func tearDown() {
	os.RemoveAll(dictDir)
}

func TestGreedyTokenizer(t *testing.T) {
	assertEqual(t, varnam.Transliterate("namaskaaram").GreedyTokenized[0].Word, "നമസ്കാരം")
	assertEqual(t, varnam.Transliterate("malayalam").GreedyTokenized[0].Word, "മലയലം")
}

func TestTokenizer(t *testing.T) {
	expected := []string{"മല", "മാല", "മള", "മലാ", "മളാ", "മാള", "മാലാ", "മാളാ"}
	for i, sug := range varnam.Transliterate("mala").Suggestions {
		assertEqual(t, sug.Word, expected[i])
	}
}

func TestLearn(t *testing.T) {
	assertEqual(t, varnam.Transliterate("malayalam").Suggestions[0].Word, "മലയലം")
	varnam.Learn("മലയാളം")
	assertEqual(t, varnam.Transliterate("malayalam").Suggestions[0].Word, "മലയാളം")
	assertEqual(t, varnam.Transliterate("malayalaththil").Suggestions[0].Word, "മലയാളത്തിൽ")
	assertEqual(t, varnam.Transliterate("malayaalar").Suggestions[0].Word, "മലയാളർ")
	assertEqual(t, varnam.Transliterate("malaykk").Suggestions[0].Word, "മലയ്ക്ക്")
}

func TestTrain(t *testing.T) {
	assertEqual(t, varnam.Transliterate("india").Suggestions[0].Word, "ഇണ്ടി")
	varnam.Train("india", "ഇന്ത്യ")
	assertEqual(t, varnam.Transliterate("india").Suggestions[0].Word, "ഇന്ത്യ")
	assertEqual(t, varnam.Transliterate("indiayil").Suggestions[0].Word, "ഇന്ത്യയിൽ")

	// Word with virama at end
	assertEqual(t, varnam.Transliterate("college").Suggestions[0].Word, "കൊല്ലെഗെ")
	varnam.Train("college", "കോളേജ്")
	assertEqual(t, varnam.Transliterate("college").Suggestions[0].Word, "കോളേജ്")
	assertEqual(t, varnam.Transliterate("collegeil").Suggestions[0].Word, "കോളേജിൽ")
	// assertEqual(t, varnam.Transliterate("collegil").Suggestions[0].Word, "കോളേജിൽ")
}

// Test zero width joiner/non-joiner things
func TestZW(t *testing.T) {
	assertEqual(t, varnam.Transliterate("thaazhvara").Suggestions[0].Word, "താഴ്വര")
	// _ is ZWNJ
	assertEqual(t, varnam.Transliterate("thaazh_vara").Suggestions[0].Word, "താഴ്‌വര")
}

func TestMain(m *testing.M) {
	setUp("ml")
	m.Run()
	tearDown()
}
