package govarnam

import (
	"context"
	"log"
	"os"
	"path"
	"strings"
	"testing"
	"time"
)

func TestMLGreedyTokenizer(t *testing.T) {
	varnam := getVarnamInstance("ml")

	assertEqual(t, varnam.TransliterateAdvanced("namaskaaram").GreedyTokenized[0].Word, "നമസ്കാരം")
	assertEqual(t, varnam.TransliterateAdvanced("malayalam").GreedyTokenized[0].Word, "മലയലം")
}

func TestMLTokenizer(t *testing.T) {
	varnam := getVarnamInstance("ml")

	// The order of this will fail if VST weights change
	expected := []string{"മല", "മള", "മലാ", "മളാ", "മാല", "മാള", "മാലാ", "മാളാ"}
	for i, sug := range varnam.TransliterateAdvanced("mala").TokenizerSuggestions {
		assertEqual(t, sug.Word, expected[i])
	}

	// TestML non lang word
	nonLangWord := varnam.TransliterateAdvanced("Шаблон")
	assertEqual(t, len(nonLangWord.ExactWords), 0)
	assertEqual(t, len(nonLangWord.ExactMatches), 0)
	assertEqual(t, len(nonLangWord.DictionarySuggestions), 0)
	assertEqual(t, len(nonLangWord.PatternDictionarySuggestions), 0)
	assertEqual(t, len(nonLangWord.TokenizerSuggestions), 1)
	assertEqual(t, len(nonLangWord.GreedyTokenized), 1)

	// TestML mixed words
	assertEqual(t, varnam.TransliterateAdvanced("naമസ്കാരmenthuNt").GreedyTokenized[0].Word, "നമസ്കാരമെന്തുണ്ട്")
	assertEqual(t, varnam.TransliterateAdvanced("*namaskaaram").GreedyTokenized[0].Word, "*നമസ്കാരം")
	assertEqual(t, varnam.TransliterateAdvanced("*nama@skaaram").GreedyTokenized[0].Word, "*നമ@സ്കാരം")
	assertEqual(t, varnam.TransliterateAdvanced("*nama@skaaram%^&").GreedyTokenized[0].Word, "*നമ@സ്കാരം%^&")

	// TestML some complex words
	assertEqual(t, varnam.TransliterateAdvanced("kambyoottar").GreedyTokenized[0].Word, "കമ്പ്യൂട്ടർ")
	assertEqual(t, varnam.TransliterateAdvanced("kambyoottar").GreedyTokenized[0].Word, "കമ്പ്യൂട്ടർ")

	// TestML fancy words
	assertEqual(t, varnam.TransliterateAdvanced("thaaaaaaaankyoo").GreedyTokenized[0].Word, "താാാാങ്ക്യൂ")

	// Test weight value
	sugs := varnam.TransliterateAdvanced("thuthuru").TokenizerSuggestions
	assertEqual(t, sugs[0].Weight, 6) // തുതുരു. Greedy. Should have highest weight
	assertEqual(t, sugs[1].Weight, 4) // തുതുറു. Last conjunct is VARNAM_MATCH_POSSIBILITY symbol
	assertEqual(t, sugs[7].Weight, 4) // തുത്തുറു. Last 2 conjuncts are VARNAM_MATCH_POSSIBILITY symbols
}

func TestMLLearn(t *testing.T) {
	varnam := getVarnamInstance("ml")

	// Non language word. Should give error
	assertEqual(t, varnam.Learn("Шаблон", 0) != nil, true)

	// Varnam will find the first word to find. Here it will be just "ഉ".
	// Since it's single conjunct, will produce an error
	assertEqual(t, varnam.Learn("ഉaള്ളിൽ", 0) != nil, true)

	assertEqual(t, varnam.Learn("Шаблонഉള്ളിൽ", 0) != nil, true)

	assertEqual(t, varnam.Learn("വ...", 0) != nil, true)

	// Before learning
	assertEqual(t, varnam.TransliterateAdvanced("malayalam").TokenizerSuggestions[0].Word, "മലയലം")

	err := varnam.Learn("മലയാളം", 0)
	checkError(err)

	// After learning
	result := varnam.TransliterateAdvanced("malayalam")
	assertEqual(t, result.ExactWords[0].Word, "മലയാളം")
	assertEqual(t, len(result.ExactMatches), 0)

	assertEqual(t, varnam.TransliterateAdvanced("malayalaththil").DictionarySuggestions[0].Word, "മലയാളത്തിൽ")
	assertEqual(t, varnam.TransliterateAdvanced("malayaalar").DictionarySuggestions[0].Word, "മലയാളർ")
	assertEqual(t, varnam.TransliterateAdvanced("malaykk").DictionarySuggestions[0].Word, "മലയ്ക്ക്")

	// Test exact matches
	result = varnam.TransliterateAdvanced("malaya")
	assertEqual(t, len(result.ExactWords), 0)
	assertEqual(t, result.ExactMatches[0].Word, "മലയ")
	assertEqual(t, result.ExactMatches[1].Word, "മലയാ")
	assertEqual(t, result.DictionarySuggestions[0].Word, "മലയാളം")
	assertEqual(t, len(result.PatternDictionarySuggestions), 0)

	start := time.Now().UTC()
	err = varnam.Learn("മലയാളത്തിൽ", 0)
	checkError(err)
	end := time.Now().UTC()

	start1SecondBefore := time.Date(start.Year(), start.Month(), start.Day(), start.Hour(), start.Minute(), start.Second()-1, 0, start.Location())
	end1SecondAfter := time.Date(end.Year(), end.Month(), end.Day(), end.Hour(), end.Minute(), end.Second()+1, 0, end.Location())

	// varnam.Debug(true)
	sugs := varnam.TransliterateAdvanced("malayala").DictionarySuggestions

	assertEqual(t, sugs[0], Suggestion{"മലയാളം", VARNAM_LEARNT_WORD_MIN_WEIGHT, sugs[0].LearnedOn})

	// Check the time learnt is right (UTC) ?
	learnedOn := time.Unix(int64(sugs[1].LearnedOn), 0)

	if !learnedOn.After(start1SecondBefore) || !learnedOn.Before(end1SecondAfter) {
		t.Errorf("Learn time %v (%v) not in between %v and %v", learnedOn, sugs[1].LearnedOn, start1SecondBefore, end1SecondAfter)
	}

	assertEqual(t, sugs[1], Suggestion{"മലയാളത്തിൽ", VARNAM_LEARNT_WORD_MIN_WEIGHT, sugs[1].LearnedOn})

	// Learn the word again
	// This word will now be at the top
	// Test if weight has increased by one now
	err = varnam.Learn("മലയാളത്തിൽ", 0)
	checkError(err)

	sug := varnam.TransliterateAdvanced("malayala").DictionarySuggestions[0]
	assertEqual(t, sug, Suggestion{"മലയാളത്തിൽ", VARNAM_LEARNT_WORD_MIN_WEIGHT + 1, sug.LearnedOn})

	// Subsequent pattern can be smaller now (no need of "thth")
	assertEqual(t, varnam.TransliterateAdvanced("malayalathil").ExactWords[0].Word, "മലയാളത്തിൽ")

	// Try words with symbols that have many possibilities
	// thu has 12 possibilties
	err = varnam.Learn("തുടങ്ങി", 0)
	checkError(err)

	assertEqual(t, varnam.TransliterateAdvanced("thudangiyittE").DictionarySuggestions[0].Word, "തുടങ്ങിയിട്ടേ")

	// Shouldn't learn single conjucnts as a word. Should give error
	assertEqual(t, varnam.Learn("കാ", 0) != nil, true)

	// Test unlearn
	varnam.Unlearn("തുടങ്ങി")
	assertEqual(t, len(varnam.TransliterateAdvanced("thudangiyittE").DictionarySuggestions), 0)
}

func TestMLTrain(t *testing.T) {
	varnam := getVarnamInstance("ml")

	assertEqual(t, varnam.TransliterateAdvanced("india").TokenizerSuggestions[0].Word, "ഇന്ദി")
	assertEqual(t, len(varnam.TransliterateAdvanced("india").PatternDictionarySuggestions), 0)

	err := varnam.Train("india", "ഇന്ത്യ")
	checkError(err)

	assertEqual(t, varnam.TransliterateAdvanced("india").ExactWords[0].Word, "ഇന്ത്യ")
	assertEqual(t, varnam.TransliterateAdvanced("ind").PatternDictionarySuggestions[0].Word, "ഇന്ത്യ")
	assertEqual(t, varnam.TransliterateAdvanced("indiayil").PatternDictionarySuggestions[0].Word, "ഇന്ത്യയിൽ")

	// Word with virama at end
	assertEqual(t, varnam.TransliterateAdvanced("college").TokenizerSuggestions[0].Word, "കൊല്ലെഗെ")
	assertEqual(t, len(varnam.TransliterateAdvanced("college").PatternDictionarySuggestions), 0)

	err = varnam.Train("college", "കോളേജ്")
	checkError(err)

	assertEqual(t, varnam.TransliterateAdvanced("college").ExactWords[0].Word, "കോളേജ്")
	assertEqual(t, varnam.TransliterateAdvanced("collegeil").PatternDictionarySuggestions[0].Word, "കോളേജിൽ")

	// TODO without e at the end
	// assertEqual(t, varnam.TransliterateAdvanced("collegil").TokenizerSuggestions[0].Word, "കോളേജിൽ")

	// Word with chil at end
	err = varnam.Train("computer", "കമ്പ്യൂട്ടർ")
	checkError(err)
	// This used to be an issue in libvarnam https://github.com/varnamproject/libvarnam/issues/166
	// GoVarnam don't have this issue because we don't use pattern_content DB for malayalam words.
	// So the problem exist for english words in pattern_content
	err = varnam.Train("kilivaathil", "കിളിവാതിൽ")
	checkError(err)

	assertEqual(t, varnam.TransliterateAdvanced("computeril").PatternDictionarySuggestions[0].Word, "കമ്പ്യൂട്ടറിൽ")
	assertEqual(t, varnam.TransliterateAdvanced("kilivaathilil").PatternDictionarySuggestions[0].Word, "കിളിവാതിലിൽ")

	err = varnam.Train("chrome", "ക്രോം")
	checkError(err)
	assertEqual(t, varnam.TransliterateAdvanced("chromeil").PatternDictionarySuggestions[0].Word, "ക്രോമിൽ")

	// Unlearning should remove pattern from DB too
	varnam.Unlearn("കോളേജ്")
	assertEqual(t, len(varnam.TransliterateAdvanced("collegeil").PatternDictionarySuggestions), 0)

	// Unlearn by pattern english
	varnam.Unlearn("computer")
	assertEqual(t, len(varnam.TransliterateAdvanced("computeril").PatternDictionarySuggestions), 0)

	err = varnam.Unlearn("computer")
	assertEqual(t, err.Error(), "nothing to unlearn")
}

func TestAnyCharacterInputWillWorkFine(t *testing.T) {
	// After working with Ruby on Rails for a while,
	// I got the habit of describing method names elaborately
	varnam := getVarnamInstance("ml")

	varnam.Learn("ഒന്നും", 0)
	varnam.Learn("പകൽ", 0)
	assertEqual(
		t,
		varnam.TransliterateAdvanced("onnum!@#$%^&*(പകൽ);'[]?.,`*/kall").DictionarySuggestions[0].Word,
		"ഒന്നും!@#$%^&*(പകൽ);'[]?.,`*ഽകല്ല്",
	)

	assertEqual(
		t,
		varnam.Transliterate("1-bi yil paTTikkunna kutti?!")[0].Word,
		"1-ബി യിൽ പഠിക്കുന്ന കുട്ടി?!",
	)
}

// TestML zero width joiner/non-joiner things
func TestMLZW(t *testing.T) {
	varnam := getVarnamInstance("ml")

	assertEqual(t, varnam.TransliterateAdvanced("thaazhvara").TokenizerSuggestions[0].Word, "താഴ്വര")
	// _ is ZWNJ
	assertEqual(t, varnam.TransliterateAdvanced("thaazh_vara").TokenizerSuggestions[0].Word, "താഴ്‌വര")

	// When _ comes after a chil in between a word, varnam explicitly generates chil. This chil won't have a ZWNJ at end
	assertEqual(t, varnam.TransliterateAdvanced("nan_ma").TokenizerSuggestions[0].Word, "നൻമ")
	assertEqual(t, varnam.TransliterateAdvanced("sam_bhavam").TokenizerSuggestions[0].Word, "സംഭവം")
}

// TestML if zwj-chils are replaced with atomic chil
func TestMLAtomicChil(t *testing.T) {
	varnam := getVarnamInstance("ml")

	err := varnam.Train("professor", "പ്രൊഫസര്‍")
	checkError(err)
	assertEqual(t, varnam.TransliterateAdvanced("professor").ExactWords[0].Word, "പ്രൊഫസർ")
}

func TestMLReverseTransliteration(t *testing.T) {
	varnam := getVarnamInstance("ml")
	oldLimit := varnam.TokenizerSuggestionsLimit
	varnam.TokenizerSuggestionsLimit = 30

	sugs, err := varnam.ReverseTransliterate("മലയാളം")
	checkError(err)

	// The order of this will fail if VST weights change
	expected := []string{"malayaaLam", "malayaaLam_", "malayALam", "malayALam_", "malayaalam", "malayaalam_", "malayAlam", "malayAlam_", "malayaLam", "malayaLam_", "malayalam", "malayalam_"}

	assertEqual(t, len(sugs), len(expected))
	for i, expectedWord := range expected {
		assertEqual(t, sugs[i].Word, expectedWord)
	}

	sugs, err = varnam.ReverseTransliterate("2019 ഏപ്രിൽ 17-ന് മലയാളം വിക്കിപീഡിയയിലെ ലേഖനങ്ങളുടെ എണ്ണം 63,000 പിന്നിട്ടു.")

	assertEqual(t, sugs[0].Word, "2019 Epril 17-n~ malayaaLam vikkipeeDiyayile lEkhanangaLuTe eNNam 63,000 pinnittu.")

	varnam.TokenizerSuggestionsLimit = oldLimit
}

func TestDictionaryLimit(t *testing.T) {
	varnam := getVarnamInstance("ml")

	words := []string{"മല", "മലയോരം", "മലയാളചലച്ചിത്രം", "മലപ്പുറം", "മലയാളത്തിൽ", "മലയ്ക്ക്", "മലയിൽ", "മലയാളം"}
	for _, word := range words {
		varnam.Learn(word, 0)
	}

	varnam.DictionarySuggestionsLimit = 2
	assertEqual(t, len(varnam.TransliterateAdvanced("mala").DictionarySuggestions), 2)

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
	assertEqual(t, len(varnam.TransliterateAdvanced("co").PatternDictionarySuggestions), 4)

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
	assertEqual(t, len(varnam.TransliterateAdvanced("editingil").PatternDictionarySuggestions), 2)
}

func TestMLLearnFromFile(t *testing.T) {
	varnam := getVarnamInstance("ml")

	// CC BY-SA 3.0 licensed
	// https://ml.wikipedia.org/wiki/National_parks_of_Taiwan
	filePath := makeFile("text.txt", "തായ്‌വാനിലെ ദേശീയോദ്യാനങ്ങൾ സംരക്ഷിതപ്രദേശങ്ങളാണ്. 7,489.49 ചതുരശ്ര കിലോമീറ്റർ (2,891.71 sq mi) വിസ്തീർണ്ണത്തിൽ വ്യാപിച്ചുകിടക്കുന്ന ഒൻപത് ദേശീയോദ്യാനങ്ങളാണ് ഇവിടെയുള്ളത്. എല്ലാ ദേശീയോദ്യാനങ്ങളും മിനിസ്ട്രി ഓഫ് ദ ഇന്റീരിയർ ഭരണത്തിൻകീഴിലാണ് നിലനിൽക്കുന്നത്. 1937-ൽ തായ്‌വാനിലെ ജാപ്പനീസ് ഭരണത്തിൻ കീഴിലായിരുന്നു ആദ്യത്തെ ദേശീയോദ്യാനം നിലവിൽവന്നത്.")

	varnam.LearnFromFile(filePath)

	assertEqual(t, len(varnam.TransliterateAdvanced("thaay_vaanile").ExactWords) != 0, true)

	// Try learning from a frequency report

	filePath = makeFile("report.txt",
		`
		നിത്യഹരിത 120
		വൃക്ഷമാണ് 89
		ഒരേയൊരു 45
		ഏഷ്യയുടെ 100
		മേലാപ്പും 12
		aadc 10
		`,
	)

	learnStatus, err := varnam.LearnFromFile(filePath)
	checkError(err)

	assertEqual(t, learnStatus.TotalWords, 6)
	assertEqual(t, learnStatus.FailedWords, 1)

	assertEqual(t, varnam.TransliterateAdvanced("nithyaharitha").ExactWords[0].Weight, 120)
	assertEqual(t, varnam.TransliterateAdvanced("melaappum").ExactWords[0].Weight, 12)
}

func TestMLTrainFromFile(t *testing.T) {
	varnam := getVarnamInstance("ml")

	// Try learning from a frequency report

	filePath := makeFile("patterns.txt",
		`
		kunnamkulam കുന്നംകുളം
		mandalamkunnu മന്ദലാംകുന്ന്
		something aadc
		`,
	)

	learnStatus, err := varnam.TrainFromFile(filePath)
	checkError(err)

	assertEqual(t, learnStatus.TotalWords, 3)
	assertEqual(t, learnStatus.FailedWords, 1)

	assertEqual(t, varnam.TransliterateAdvanced("mandalamkunnu").ExactWords[0].Word, "മന്ദലാംകുന്ന്")
	assertEqual(t, len(varnam.TransliterateAdvanced("something").ExactWords), 0)
}

func TestMLExportAndImport(t *testing.T) {
	varnam := getVarnamInstance("ml")

	words := []WordInfo{
		WordInfo{0, "മനുഷ്യൻ", 0, 0},
		WordInfo{0, "മണ്ഡലം", 0, 0},
		WordInfo{0, "മിലാൻ", 0, 0},
	}

	varnam.LearnMany(words)

	exportFileIntendedPath := path.Join(testTempDir, "export")
	exportFilePath := exportFileIntendedPath + "-1.vlf"

	varnam.Export(exportFileIntendedPath, 300)

	// read the whole file at once
	b, err := os.ReadFile(exportFilePath)
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
		results := varnam.searchDictionary(context.Background(), []string{wordInfo.word}, searchMatches)

		assertEqual(t, len(results) > 0, true)
	}

	// Test if importing file with a JSON work
	filePath := makeFile("custom.json", `
		{
			"words": [
				{
					"w": "അൾജീരിയ",
					"c": 25,
					"l": 1531131220
				}
			],
			"patterns": [
				{
					"p": "algeria",
					"w": "അൾജീരിയ"
				}
			]
		}
	`)
	varnam.Import(filePath)

	assertEqual(t, varnam.TransliterateAdvanced("algeria").ExactWords[0], Suggestion{
		Word:      "അൾജീരിയ",
		Weight:    VARNAM_LEARNT_WORD_MIN_WEIGHT + 25,
		LearnedOn: 1531131220,
	})
}

func TestMLSearchSymbolTable(t *testing.T) {
	varnam := getVarnamInstance("ml")

	search := NewSearchSymbol()
	search.Value1 = "ക"
	results, err := varnam.SearchSymbolTable(context.Background(), search)
	checkError(err)

	for _, result := range results {
		assertEqual(t, result.Value1, "ക")
	}

	search.Value1 = "LIKE ക%"
	search.Pattern = "ka"
	results, err = varnam.SearchSymbolTable(context.Background(), search)
	checkError(err)

	assertEqual(t, results[0].Value1, "ക")
	assertEqual(t, results[1].Value1, "കാ")
}

func TestMLDictionaryMatchExact(t *testing.T) {
	varnam := getVarnamInstance("ml")

	varnam.DictionaryMatchExact = true

	varnam.Learn("പനിയിൽ", 0)
	varnam.Learn("പണിയിൽ", 0)

	result := varnam.TransliterateAdvanced("pani")
	assertEqual(t, len(result.DictionarySuggestions), 1)

	varnam.DictionaryMatchExact = false
}

func TestMLRecentlyLearnedWords(t *testing.T) {
	varnam := getVarnamInstance("ml")

	words := []string{"ആലപ്പുഴ", "എറണാകുളം", "തൃശ്ശൂർ", "പാലക്കാട്", "കോഴിക്കോട്"}
	for _, word := range words {
		varnam.Learn(word, 0)
	}

	result, err := varnam.GetRecentlyLearntWords(context.Background(), 0, len(words))
	checkError(err)

	assertEqual(t, len(result), len(words))
	log.Println(result, words)
	for i, sug := range result {
		assertEqual(t, sug.Word, words[len(words)-i-1])
	}

	result, err = varnam.GetRecentlyLearntWords(context.Background(), 4, len(words))
	assertEqual(t, result[0].Word, "ആലപ്പുഴ")
}

func TestMLGetSuggestions(t *testing.T) {
	varnam := getVarnamInstance("ml")

	words := []string{"ആലപ്പുഴ", "ആലം", "ആലാപനം"}
	for _, word := range words {
		varnam.Learn(word, 0)
	}

	varnam.DictionarySuggestionsLimit = 5
	result := varnam.GetSuggestions(context.Background(), "ആല")
	assertEqual(t, len(result), 3)

	assertEqual(t, result[0].Word, "ആലപ്പുഴ")
}
