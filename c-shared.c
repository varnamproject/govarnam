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

void destroySuggestion(void* pointer)
{
  if (pointer != NULL) {
    Suggestion* sug = (Suggestion*) pointer;
    free(sug->Word);
    sug->Word = NULL;
    free(sug);
    sug = NULL;
  }
}

void destroyTransliterationResult(TransliterationResult* result)
{
  varray_free(result->ExactMatch, &destroySuggestion);
  varray_free(result->Suggestions, &destroySuggestion);
  varray_free(result->GreedyTokenized, &destroySuggestion);
  result->ExactMatch = NULL;
  result->Suggestions = NULL;
  result->GreedyTokenized = NULL;
  free(result);
  result = NULL;
}
