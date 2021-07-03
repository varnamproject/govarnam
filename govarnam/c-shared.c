#include "c-shared.h"
#include "stdio.h"
#include "stdlib.h"
#include "c-shared-varray.h"

void callTransliterateCallback(TransliterateCallbackFn func, varray* exact_match, varray* suggestions, varray* greedy_tokenized, int dictionary_result_count)
{
  TransliterationResult *result = (TransliterationResult*) malloc (sizeof(TransliterationResult));
  result->ExactMatch = exact_match;
  result->Suggestions = suggestions;
  result->GreedyTokenized = greedy_tokenized;

  func(result);
}

extern Suggestion* makeSuggestion(char* word, int weight, int learned_on)
{
  Suggestion *sug = (Suggestion*) malloc (sizeof(Suggestion));
  sug->Word = word;
  sug->Weight = weight;
  sug->LearnedOn = learned_on;
  return sug;
}
