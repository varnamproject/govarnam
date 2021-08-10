package govarnam

import (
	"context"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"testing"
	"time"
)

func TestMLGreedyTokenizer(t *testing.T) {
	varnam := getVarnamInstance("ml")

	assertEqual(t, varnam.Transliterate("namaskaaram").GreedyTokenized[0].Word, "നമസ്കാരം")
	assertEqual(t, varnam.Transliterate("malayalam").GreedyTokenized[0].Word, "മലയലം")
}

func TestMLTokenizer(t *testing.T) {
	varnam := getVarnamInstance("ml")

	// The order of this will fail if VST weights change
	expected := []string{"മല", "മള", "മലാ", "മളാ", "മാല", "മാള", "മാലാ", "മാളാ"}
	for i, sug := range varnam.Transliterate("mala").TokenizerSuggestions {
		assertEqual(t, sug.Word, expected[i])
	}

	// TestML non lang word
	nonLangWord := varnam.Transliterate("Шаблон")
	assertEqual(t, len(nonLangWord.ExactMatches), 0)
	assertEqual(t, len(nonLangWord.DictionarySuggestions), 0)
	assertEqual(t, len(nonLangWord.PatternDictionarySuggestions), 0)
	assertEqual(t, len(nonLangWord.TokenizerSuggestions), 1)
	assertEqual(t, len(nonLangWord.GreedyTokenized), 1)

	// TestML mixed words
	assertEqual(t, varnam.Transliterate("naമസ്കാരmenthuNt").GreedyTokenized[0].Word, "നമസ്കാരമെന്തുണ്ട്")
	assertEqual(t, varnam.Transliterate("*namaskaaram").GreedyTokenized[0].Word, "*നമസ്കാരം")
	assertEqual(t, varnam.Transliterate("*nama@skaaram").GreedyTokenized[0].Word, "*നമ@സ്കാരം")
	assertEqual(t, varnam.Transliterate("*nama@skaaram%^&").GreedyTokenized[0].Word, "*നമ@സ്കാരം%^&")

	// TestML some complex words
	assertEqual(t, varnam.Transliterate("kambyoottar").GreedyTokenized[0].Word, "കമ്പ്യൂട്ടർ")
	assertEqual(t, varnam.Transliterate("kambyoottar").GreedyTokenized[0].Word, "കമ്പ്യൂട്ടർ")

	// TestML fancy words
	assertEqual(t, varnam.Transliterate("thaaaaaaaankyoo").GreedyTokenized[0].Word, "താാാാങ്ക്യൂ")

	// Test weight value
	sugs := varnam.Transliterate("thuthuru").TokenizerSuggestions
	assertEqual(t, sugs[0].Weight, 6) // തുതുരു. Greedy. Should have highest weight
	assertEqual(t, sugs[1].Weight, 4) // തുതുറു. Last conjunct is VARNAM_MATCH_POSSIBILITY symbol
	assertEqual(t, sugs[7].Weight, 4) // തുത്തുറു. Last 2 conjuncts are VARNAM_MATCH_POSSIBILITY symbols
}

func TestMLLearn(t *testing.T) {
	varnam := getVarnamInstance("ml")

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

	assertEqual(t, sugs[0], Suggestion{"മലയാളം", VARNAM_LEARNT_WORD_MIN_WEIGHT, sugs[0].LearnedOn})

	// Check the time learnt is right (UTC) ?
	learnedOn := time.Unix(int64(sugs[1].LearnedOn), 0)

	if !learnedOn.After(start1SecondBefore) || !learnedOn.Before(end1SecondAfter) {
		t.Errorf("Learn time %v (%v) not in between %v and %v", learnedOn, sugs[1].LearnedOn, start1SecondBefore, end1SecondAfter)
	}

	assertEqual(t, sugs[1], Suggestion{"മലയാളത്തിൽ", VARNAM_LEARNT_WORD_MIN_WEIGHT, sugs[1].LearnedOn})

	// Learn the word again
	// This word will now be at the top
	// TestML if weight has increased by one now
	err = varnam.Learn("മലയാളത്തിൽ", 0)
	checkError(err)

	sug := varnam.Transliterate("malayala").DictionarySuggestions[0]
	assertEqual(t, sug, Suggestion{"മലയാളത്തിൽ", VARNAM_LEARNT_WORD_MIN_WEIGHT + 1, sug.LearnedOn})

	// Subsequent pattern can be smaller now (no need of "thth")
	assertEqual(t, varnam.Transliterate("malayalathil").ExactMatches[0].Word, "മലയാളത്തിൽ")

	// Try words with symbols that have many possibilities
	// thu has 12 possibilties
	err = varnam.Learn("തുടങ്ങി", 0)
	checkError(err)

	assertEqual(t, varnam.Transliterate("thudangiyittE").DictionarySuggestions[0].Word, "തുടങ്ങിയിട്ടേ")

	// Shouldn't learn single conjucnts as a word. Should give error
	assertEqual(t, varnam.Learn("കാ", 0) != nil, true)

	// Test unlearn
	varnam.Unlearn("തുടങ്ങി")
	assertEqual(t, len(varnam.Transliterate("thudangiyittE").DictionarySuggestions), 0)
}

func TestMLTrain(t *testing.T) {
	varnam := getVarnamInstance("ml")

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

	// Word with chil at end
	err = varnam.Train("computer", "കമ്പ്യൂട്ടർ")
	checkError(err)
	// This used to be an issue in libvarnam https://github.com/varnamproject/libvarnam/issues/166
	// GoVarnam don't have this issue because we don't use pattern_content DB for malayalam words.
	// So the problem exist for english words in pattern_content
	err = varnam.Train("kilivaathil", "കിളിവാതിൽ")
	checkError(err)

	assertEqual(t, varnam.Transliterate("computeril").PatternDictionarySuggestions[0].Word, "കമ്പ്യൂട്ടറിൽ")
	assertEqual(t, varnam.Transliterate("kilivaathilil").PatternDictionarySuggestions[0].Word, "കിളിവാതിലിൽ")

	// Unlearning should remove pattern from DB too
	varnam.Unlearn("കോളേജ്")
	assertEqual(t, len(varnam.Transliterate("collegeil").PatternDictionarySuggestions), 0)

	// Unlearn by pattern english
	varnam.Unlearn("computer")
	assertEqual(t, len(varnam.Transliterate("computeril").PatternDictionarySuggestions), 0)

	err = varnam.Unlearn("computer")
	assertEqual(t, err.Error(), "nothing to unlearn")
}

// TestML zero width joiner/non-joiner things
func TestMLZW(t *testing.T) {
	varnam := getVarnamInstance("ml")

	assertEqual(t, varnam.Transliterate("thaazhvara").TokenizerSuggestions[0].Word, "താഴ്വര")
	// _ is ZWNJ
	assertEqual(t, varnam.Transliterate("thaazh_vara").TokenizerSuggestions[0].Word, "താഴ്‌വര")

	// When _ comes after a chil in between a word, varnam explicitly generates chil. This chil won't have a ZWNJ at end
	assertEqual(t, varnam.Transliterate("nan_ma").TokenizerSuggestions[0].Word, "നൻമ")
	assertEqual(t, varnam.Transliterate("sam_bhavam").TokenizerSuggestions[0].Word, "സംഭവം")
}

// TestML if zwj-chils are replaced with atomic chil
func TestMLAtomicChil(t *testing.T) {
	varnam := getVarnamInstance("ml")

	varnam.Train("professor", "പ്രൊഫസര്‍")
	assertEqual(t, varnam.Transliterate("professor").ExactMatches[0].Word, "പ്രൊഫസർ")
}

func TestMLReverseTransliteration(t *testing.T) {
	varnam := getVarnamInstance("ml")

	sugs, err := varnam.ReverseTransliterate("മലയാളം")
	checkError(err)

	// The order of this will fail if VST weights change
	expected := []string{"malayaaLam", "malayALam", "malayaalam", "malayAlam", "malayaLam", "malayalam"}
	for i, sug := range sugs {
		assertEqual(t, sug.Word, expected[i])
	}

	sugs, err = varnam.ReverseTransliterate("2019 ഏപ്രിൽ 17-ന് മലയാളം വിക്കിപീഡിയയിലെ ലേഖനങ്ങളുടെ എണ്ണം 63,000 പിന്നിട്ടു.")

	assertEqual(t, sugs[0].Word, "2019 Epril 17-n~ malayaaLam vikkipeeDiyayile lEkhanaNGaLuTe eNNam 63,000 pinnittu.")
}

func TestDictionaryLimit(t *testing.T) {
	varnam := getVarnamInstance("ml")

	words := []string{"മല", "മലയോരം", "മലയാളചലച്ചിത്രം", "മലപ്പുറം", "മലയാളത്തിൽ", "മലയ്ക്ക്", "മലയിൽ", "മലയാളം"}
	for _, word := range words {
		varnam.Learn(word, 0)
	}

	varnam.DictionarySuggestionsLimit = 2
	assertEqual(t, len(varnam.Transliterate("mala").DictionarySuggestions), 2)

	patternsAndWords := map[string]string{
		"collateral": "കോലാറ്ററൽ",
		"collective": "കളക്ടീവ്",
		"collector":  "കളക്ടർ",
		"college":    "കോളേജ്",
		"colombia":   "കൊളംബിയ",
		"commons":    "കോമൺസ്",
	}
	for pattern, word := range patternsAndWords {
		varnam.Train(pattern, word)
	}

	varnam.PatternDictionarySuggestionsLimit = 4
	assertEqual(t, len(varnam.Transliterate("co").PatternDictionarySuggestions), 4)

	// Test multiple matching words while partializing
	patternsAndWords = map[string]string{
		"edit":    "എഡിറ്റ്",
		"editing": "എഡിറ്റിംഗ്",
		"edition": "എഡിഷൻ",
	}
	for pattern, word := range patternsAndWords {
		varnam.Train(pattern, word)
	}

	varnam.PatternDictionarySuggestionsLimit = 2

	// Tokenizer will work on 2 words: എഡിറ്റ് & എഡിറ്റിംഗ്
	// Total results = 4+
	assertEqual(t, len(varnam.Transliterate("editingil").PatternDictionarySuggestions), 2)
}

func TestMLLearnFromFile(t *testing.T) {
	varnam := getVarnamInstance("ml")

	filePath := path.Join(testTempDir, "text.txt")

	file, err := os.Create(filePath)
	if err != nil {
		return
	}
	defer file.Close()

	// CC BY-SA 3.0 licensed
	// https://ml.wikipedia.org/wiki/National_parks_of_Taiwan
	file.WriteString("തായ്‌വാനിലെ ദേശീയോദ്യാനങ്ങൾ സംരക്ഷിതപ്രദേശങ്ങളാണ്. 7,489.49 ചതുരശ്ര കിലോമീറ്റർ (2,891.71 sq mi) വിസ്തീർണ്ണത്തിൽ വ്യാപിച്ചുകിടക്കുന്ന ഒൻപത് ദേശീയോദ്യാനങ്ങളാണ് ഇവിടെയുള്ളത്. എല്ലാ ദേശീയോദ്യാനങ്ങളും മിനിസ്ട്രി ഓഫ് ദ ഇന്റീരിയർ ഭരണത്തിൻകീഴിലാണ് നിലനിൽക്കുന്നത്. 1937-ൽ തായ്‌വാനിലെ ജാപ്പനീസ് ഭരണത്തിൻ കീഴിലായിരുന്നു ആദ്യത്തെ ദേശീയോദ്യാനം നിലവിൽവന്നത്.")

	varnam.LearnFromFile(filePath)

	assertEqual(t, len(varnam.Transliterate("thaay_vaanile").ExactMatches) != 0, true)

	// Try learning from a frequency report

	filePath = path.Join(testTempDir, "report.txt")

	file, err = os.Create(filePath)
	if err != nil {
		return
	}
	defer file.Close()

	file.WriteString(`നിത്യഹരിത 120
	വൃക്ഷമാണ് 89
	ഒരേയൊരു 45
	ഏഷ്യയുടെ 100
	മേലാപ്പും 12
	aadc 10`)

	learnStatus, err := varnam.LearnFromFile(filePath)
	checkError(err)

	assertEqual(t, learnStatus.TotalWords, 6)
	assertEqual(t, learnStatus.FailedWords, 1)

	assertEqual(t, varnam.Transliterate("nithyaharitha").ExactMatches[0].Weight, 120)
	assertEqual(t, varnam.Transliterate("melaappum").ExactMatches[0].Weight, 12)
}

func TestMLExportAndImport(t *testing.T) {
	varnam := getVarnamInstance("ml")

	words := []WordInfo{
		WordInfo{0, "മനുഷ്യൻ", 0, 0},
		WordInfo{0, "മണ്ഡലം", 0, 0},
		WordInfo{0, "മിലാൻ", 0, 0},
	}

	varnam.LearnMany(words)

	exportFilePath := path.Join(testTempDir, "export")

	varnam.Export(exportFilePath)

	// read the whole file at once
	b, err := ioutil.ReadFile(exportFilePath)
	if err != nil {
		panic(err)
	}
	exportFileContents := string(b)

	for _, wordInfo := range words {
		assertEqual(t, strings.Contains(exportFileContents, wordInfo.word), true)

		// Unlearn so that we can import next
		varnam.Unlearn(wordInfo.word)
	}

	varnam.Import(exportFilePath)

	for _, wordInfo := range words {
		results := varnam.searchDictionary(context.Background(), []string{wordInfo.word}, false)

		assertEqual(t, len(results) > 0, true)
	}
}
