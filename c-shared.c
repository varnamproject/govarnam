#include "c-shared.h"
#include "stdio.h"
#include "stdlib.h"
#include "c-shared-varray.h"

Suggestion* makeSuggestion(char* word, int weight, int learned_on)
{
  Suggestion *sug = (Suggestion*) malloc (sizeof(Suggestion));
  sug->Word = word;
  sug->Weight = weight;
  sug->LearnedOn = learned_on;
  printf("ccc");
  return sug;
}

TransliterationResult* makeResult(varray* exact_match, varray* suggestions, varray* greedy_tokenized, int dictionary_result_count)
{
  TransliterationResult *result = (TransliterationResult*) malloc (sizeof(TransliterationResult));
  result->ExactMatch = exact_match;
  result->Suggestions = suggestions;
  result->GreedyTokenized = greedy_tokenized;
  result->DictionaryResultCount = dictionary_result_count;
  return result;
}
