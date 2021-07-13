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
	"time"
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

func checkError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func setUp(schemeID string) {
	_, filename, _, _ := runtime.Caller(0)
	projectRoot := path.Join(path.Dir(filename), "..")

	vstLoc := path.Join(projectRoot, "schemes", schemeID+".vst")

	dictDir, err := ioutil.TempDir("", "govarnam_test")
	if err != nil {
		log.Fatal(err)
	}

	dictLoc := path.Join(dictDir, schemeID+".vst.learnings")
	err = makeDictionary(dictLoc)
	if err != nil {
		log.Fatal(err)
	}

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
	// The order of this will fail if VST weights change
	expected := []string{"മല", "മള", "മലാ", "മളാ", "മാല", "മാള", "മാലാ", "മാളാ"}
	for i, sug := range varnam.Transliterate("mala").TokenizerSuggestions {
		assertEqual(t, sug.Word, expected[i])
	}

	// Test non lang word
	nonLangWord := varnam.Transliterate("Шаблон")
	assertEqual(t, len(nonLangWord.ExactMatches), 0)
	assertEqual(t, len(nonLangWord.DictionarySuggestions), 0)
	assertEqual(t, len(nonLangWord.PatternDictionarySuggestions), 0)
	assertEqual(t, len(nonLangWord.TokenizerSuggestions), 1)
	assertEqual(t, len(nonLangWord.GreedyTokenized), 1)

	// Test mixed words
	assertEqual(t, varnam.Transliterate("naമസ്കാരmenthuNt").GreedyTokenized[0].Word, "നമസ്കാരമെന്തുണ്ട്")
	assertEqual(t, varnam.Transliterate("*namaskaaram").GreedyTokenized[0].Word, "*നമസ്കാരം")
	assertEqual(t, varnam.Transliterate("*nama@skaaram").GreedyTokenized[0].Word, "*നമ@സ്കാരം")
	assertEqual(t, varnam.Transliterate("*nama@skaaram%^&").GreedyTokenized[0].Word, "*നമ@സ്കാരം%^&")

	// Test some complex words
	assertEqual(t, varnam.Transliterate("kambyoottar").GreedyTokenized[0].Word, "കമ്പ്യൂട്ടർ")
	assertEqual(t, varnam.Transliterate("kambyoottar").GreedyTokenized[0].Word, "കമ്പ്യൂട്ടർ")

	// Test fancy words
	assertEqual(t, varnam.Transliterate("thaaaaaaaankyoo").GreedyTokenized[0].Word, "താാാാങ്ക്യൂ")
}

func TestLearn(t *testing.T) {
	// Non language word. Should give error
	assertEqual(t, varnam.Learn("Шаблон", 0) != nil, true)

	// Before learning
	assertEqual(t, varnam.Transliterate("malayalam").TokenizerSuggestions[0].Word, "മലയലം")

	err := varnam.Learn("മലയാളം", 0)
	checkError(err)

	// After learning
	assertEqual(t, varnam.Transliterate("malayalam").ExactMatches[0].Word, "മലയാളം")
	assertEqual(t, varnam.Transliterate("malayalaththil").DictionarySuggestions[0].Word, "മലയാളത്തിൽ")
	assertEqual(t, varnam.Transliterate("malayaalar").DictionarySuggestions[0].Word, "മലയാളർ")
	assertEqual(t, varnam.Transliterate("malaykk").DictionarySuggestions[0].Word, "മലയ്ക്ക്")

	start := time.Now().UTC()
	err = varnam.Learn("മലയാളത്തിൽ", 0)
	checkError(err)
	end := time.Now().UTC()

	start1SecondBefore := time.Date(start.Year(), start.Month(), start.Day(), start.Hour(), start.Minute(), start.Second()-1, 0, start.Location())
	end1SecondAfter := time.Date(end.Year(), end.Month(), end.Day(), end.Hour(), end.Minute(), end.Second()+1, 0, end.Location())

	// varnam.Debug(true)
	sugs := varnam.Transliterate("malayala").DictionarySuggestions

	assertEqual(t, sugs[0], Suggestion{"മലയാളം", VARNAM_LEARNT_WORD_MIN_CONFIDENCE, sugs[0].LearnedOn})

	// Check the time learnt is right (UTC) ?
	learnedOn := time.Unix(int64(sugs[1].LearnedOn), 0)

	if !learnedOn.After(start1SecondBefore) || !learnedOn.Before(end1SecondAfter) {
		t.Errorf("Learn time %v (%v) not in between %v and %v", learnedOn, sugs[1].LearnedOn, start1SecondBefore, end1SecondAfter)
	}

	assertEqual(t, sugs[1], Suggestion{"മലയാളത്തിൽ", VARNAM_LEARNT_WORD_MIN_CONFIDENCE, sugs[1].LearnedOn})

	// Learn the word again
	// This word will now be at the top
	// Test if confidence has increased by one now
	err = varnam.Learn("മലയാളത്തിൽ", 0)
	checkError(err)

	sug := varnam.Transliterate("malayala").DictionarySuggestions[0]
	assertEqual(t, sug, Suggestion{"മലയാളത്തിൽ", VARNAM_LEARNT_WORD_MIN_CONFIDENCE + 1, sug.LearnedOn})

	// Subsequent pattern can be smaller now (no need of "thth")
	assertEqual(t, varnam.Transliterate("malayalathil").ExactMatches[0].Word, "മലയാളത്തിൽ")
}

func TestTrain(t *testing.T) {
	assertEqual(t, varnam.Transliterate("india").TokenizerSuggestions[0].Word, "ഇന്ദി")
	assertEqual(t, len(varnam.Transliterate("india").PatternDictionarySuggestions), 0)

	err := varnam.Train("india", "ഇന്ത്യ")
	checkError(err)

	assertEqual(t, varnam.Transliterate("india").ExactMatches[0].Word, "ഇന്ത്യ")
	assertEqual(t, varnam.Transliterate("indiayil").PatternDictionarySuggestions[0].Word, "ഇന്ത്യയിൽ")

	// Word with virama at end
	assertEqual(t, varnam.Transliterate("college").TokenizerSuggestions[0].Word, "കൊല്ലെഗെ")
	assertEqual(t, len(varnam.Transliterate("college").PatternDictionarySuggestions), 0)

	err = varnam.Train("college", "കോളേജ്")
	checkError(err)

	assertEqual(t, varnam.Transliterate("college").ExactMatches[0].Word, "കോളേജ്")
	assertEqual(t, varnam.Transliterate("collegeil").PatternDictionarySuggestions[0].Word, "കോളേജിൽ")

	// TODO without e at the end
	// assertEqual(t, varnam.Transliterate("collegil").TokenizerSuggestions[0].Word, "കോളേജിൽ")
}

// Test zero width joiner/non-joiner things
func TestZW(t *testing.T) {
	assertEqual(t, varnam.Transliterate("thaazhvara").TokenizerSuggestions[0].Word, "താഴ്വര")
	// _ is ZWNJ
	assertEqual(t, varnam.Transliterate("thaazh_vara").TokenizerSuggestions[0].Word, "താഴ്‌വര")

	// When _ comes after a chil, varnam explicitly generates chil without ZWNJ at end
	assertEqual(t, varnam.Transliterate("n_").TokenizerSuggestions[0].Word, "ൻ")
	assertEqual(t, varnam.Transliterate("nan_ma").TokenizerSuggestions[0].Word, "നൻമ")
	assertEqual(t, varnam.Transliterate("sam_bhavam").TokenizerSuggestions[0].Word, "സംഭവം")
}

// Test if zwj-chils are replaced with atomic chil
func TestAtomicChil(t *testing.T) {
	varnam.Train("professor", "പ്രൊഫസര്‍")
	assertEqual(t, varnam.Transliterate("professor").ExactMatches[0].Word, "പ്രൊഫസർ")
}

func TestMain(m *testing.M) {
	setUp("ml")
	m.Run()
	tearDown()
}
