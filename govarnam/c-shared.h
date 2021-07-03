#ifndef __C_SHARED_H__
#define __C_SHARED_H__

#include "c-shared-varray.h"

typedef struct Suggestion_t {
  char* Word;
  int Weight;
  int LearnedOn;
} Suggestion;

typedef struct TransliterationResult_t {
  varray* ExactMatch;
  varray* Suggestions;
  varray* GreedyTokenized;
  int DictionaryResultCount;
} TransliterationResult;

typedef TransliterationResult TransliterationResult;

typedef void (*TransliterateCallbackFn)(TransliterationResult* result);

void callTransliterateCallback(TransliterateCallbackFn func, varray* exact_match, varray* suggestions, varray* greedy_tokenized, int dictionary_result_count);

extern  Suggestion* makeSuggestion(char* word, int weight, int learned_on);

#endif /* __C_SHARED_H__ */