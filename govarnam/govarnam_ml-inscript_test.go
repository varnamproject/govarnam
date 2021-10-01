package govarnam

import (
	"testing"
)

// Inscript = Inscript 2

func TestMLInscriptGreedyTokenizer(t *testing.T) {
	varnam := getVarnamInstance("ml-inscript")

	assertEqual(t, varnam.TransliterateAdvanced("ECs").GreedyTokenized[0].Word, "ആണേ")
	assertEqual(t, varnam.TransliterateAdvanced("Zhdha").GreedyTokenized[0].Word, "എപ്പോ")
}

func TestMLInscriptTokenizer(t *testing.T) {
	varnam := getVarnamInstance("ml-inscript")

	// TestMLInscript non lang word
	nonLangWord := varnam.TransliterateAdvanced("Шаблон")
	assertEqual(t, len(nonLangWord.ExactMatches), 0)
	assertEqual(t, len(nonLangWord.DictionarySuggestions), 0)
	assertEqual(t, len(nonLangWord.PatternDictionarySuggestions), 0)
	assertEqual(t, len(nonLangWord.TokenizerSuggestions), 1)
	assertEqual(t, len(nonLangWord.GreedyTokenized), 1)

	// TestMLInscript mixed words & symbol escapes with |
	assertEqual(t, varnam.TransliterateAdvanced(";aയ്ച്ചാclf").GreedyTokenized[0].Word, "ചോയ്ച്ചാമതി")
	assertEqual(t, varnam.TransliterateAdvanced("|*vgcdc").GreedyTokenized[0].Word, "*നുമ്മ")
	assertEqual(t, varnam.TransliterateAdvanced("|*vg@cdc").GreedyTokenized[0].Word, "*നു@മ്മ")
	assertEqual(t, varnam.TransliterateAdvanced("|;|\"||*|'|-|>|\\^4^k").GreedyTokenized[0].Word, ";\"|*'->\\₹^ക")

	// TestMLInscript some complex words
	assertEqual(t, varnam.TransliterateAdvanced("Gh/aif;d;g").GreedyTokenized[0].Word, "ഉപയോഗിച്ചു")
	assertEqual(t, varnam.TransliterateAdvanced(";a/d;d;eclf").GreedyTokenized[0].Word, "ചോയ്ച്ചാമതി")

	// TestMLInscript fancy words
	assertEqual(t, varnam.TransliterateAdvanced("leeeeUdkd/t").GreedyTokenized[0].Word, "താാാാങ്ക്യൂ")
}

// func TestMLInscriptLearn(t *testing.T) {
// 	varnam := getVarnamInstance("ml-inscript")

// 	// Non language word. Should give error
// 	assertEqual(t, varnam.Learn("Шаблон", 0) != nil, true)

// 	// Before learning
// 	assertEqual(t, varnam.TransliterateAdvanced("malayalam").TokenizerSuggestions[0].Word, "മലയലം")

// 	err := varnam.Learn("മലയാളം", 0)
// 	checkError(err)

// 	// After learning
// 	assertEqual(t, varnam.TransliterateAdvanced("malayalam").ExactMatches[0].Word, "മലയാളം")
// 	assertEqual(t, varnam.TransliterateAdvanced("malayalaththil").DictionarySuggestions[0].Word, "മലയാളത്തിൽ")
// 	assertEqual(t, varnam.TransliterateAdvanced("malayaalar").DictionarySuggestions[0].Word, "മലയാളർ")
// 	assertEqual(t, varnam.TransliterateAdvanced("malaykk").DictionarySuggestions[0].Word, "മലയ്ക്ക്")

// 	start := time.Now().UTC()
// 	err = varnam.Learn("മലയാളത്തിൽ", 0)
// 	checkError(err)
// 	end := time.Now().UTC()

// 	start1SecondBefore := time.Date(start.Year(), start.Month(), start.Day(), start.Hour(), start.Minute(), start.Second()-1, 0, start.Location())
// 	end1SecondAfter := time.Date(end.Year(), end.Month(), end.Day(), end.Hour(), end.Minute(), end.Second()+1, 0, end.Location())

// 	// varnam.Debug(true)
// 	sugs := varnam.TransliterateAdvanced("malayala").DictionarySuggestions

// 	assertEqual(t, sugs[0], Suggestion{"മലയാളം", VARNAM_LEARNT_WORD_MIN_WEIGHT, sugs[0].LearnedOn})

// 	// Check the time learnt is right (UTC) ?
// 	learnedOn := time.Unix(int64(sugs[1].LearnedOn), 0)

// 	if !learnedOn.After(start1SecondBefore) || !learnedOn.Before(end1SecondAfter) {
// 		t.Errorf("Learn time %v (%v) not in between %v and %v", learnedOn, sugs[1].LearnedOn, start1SecondBefore, end1SecondAfter)
// 	}

// 	assertEqual(t, sugs[1], Suggestion{"മലയാളത്തിൽ", VARNAM_LEARNT_WORD_MIN_WEIGHT, sugs[1].LearnedOn})

// 	// Learn the word again
// 	// This word will now be at the top
// 	// TestMLInscript if weight has increased by one now
// 	err = varnam.Learn("മലയാളത്തിൽ", 0)
// 	checkError(err)

// 	sug := varnam.TransliterateAdvanced("malayala").DictionarySuggestions[0]
// 	assertEqual(t, sug, Suggestion{"മലയാളത്തിൽ", VARNAM_LEARNT_WORD_MIN_WEIGHT + 1, sug.LearnedOn})

// 	// Subsequent pattern can be smaller now (no need of "thth")
// 	assertEqual(t, varnam.TransliterateAdvanced("malayalathil").ExactMatches[0].Word, "മലയാളത്തിൽ")

// 	// Try words with symbols that have many possibilities
// 	// thu has 12 possibilties
// 	err = varnam.Learn("തുടങ്ങി", 0)
// 	checkError(err)

// 	assertEqual(t, varnam.TransliterateAdvanced("thudangiyittE").DictionarySuggestions[0].Word, "തുടങ്ങിയിട്ടേ")
// }

// func TestMLInscriptTrain(t *testing.T) {
// 	varnam := getVarnamInstance("ml-inscript")

// 	assertEqual(t, varnam.TransliterateAdvanced("india").TokenizerSuggestions[0].Word, "ഇന്ദി")
// 	assertEqual(t, len(varnam.TransliterateAdvanced("india").PatternDictionarySuggestions), 0)

// 	err := varnam.Train("india", "ഇന്ത്യ")
// 	checkError(err)

// 	assertEqual(t, varnam.TransliterateAdvanced("india").ExactMatches[0].Word, "ഇന്ത്യ")
// 	assertEqual(t, varnam.TransliterateAdvanced("indiayil").PatternDictionarySuggestions[0].Word, "ഇന്ത്യയിൽ")

// 	// Word with virama at end
// 	assertEqual(t, varnam.TransliterateAdvanced("college").TokenizerSuggestions[0].Word, "കൊല്ലെഗെ")
// 	assertEqual(t, len(varnam.TransliterateAdvanced("college").PatternDictionarySuggestions), 0)

// 	err = varnam.Train("college", "കോളേജ്")
// 	checkError(err)

// 	assertEqual(t, varnam.TransliterateAdvanced("college").ExactMatches[0].Word, "കോളേജ്")
// 	assertEqual(t, varnam.TransliterateAdvanced("collegeil").PatternDictionarySuggestions[0].Word, "കോളേജിൽ")

// 	// TODO without e at the end
// 	// assertEqual(t, varnam.TransliterateAdvanced("collegil").TokenizerSuggestions[0].Word, "കോളേജിൽ")
// }

// // TestMLInscript zero width joiner/non-joiner things
// func TestMLInscriptZW(t *testing.T) {
// 	varnam := getVarnamInstance("ml-inscript")

// 	assertEqual(t, varnam.TransliterateAdvanced("thaazhvara").TokenizerSuggestions[0].Word, "താഴ്വര")
// 	// _ is ZWNJ
// 	assertEqual(t, varnam.TransliterateAdvanced("thaazh_vara").TokenizerSuggestions[0].Word, "താഴ്‌വര")

// 	// When _ comes after a chil, varnam explicitly generates chil without ZWNJ at end
// 	assertEqual(t, varnam.TransliterateAdvanced("n_").TokenizerSuggestions[0].Word, "ൻ")
// 	assertEqual(t, varnam.TransliterateAdvanced("nan_ma").TokenizerSuggestions[0].Word, "നൻമ")
// 	assertEqual(t, varnam.TransliterateAdvanced("sam_bhavam").TokenizerSuggestions[0].Word, "സംഭവം")
// }

// // TestMLInscript if zwj-chils are replaced with atomic chil
// func TestMLInscriptAtomicChil(t *testing.T) {
// 	varnam := getVarnamInstance("ml-inscript")

// 	varnam.Train("professor", "പ്രൊഫസര്‍")
// 	assertEqual(t, varnam.TransliterateAdvanced("professor").ExactMatches[0].Word, "പ്രൊഫസർ")
// }
