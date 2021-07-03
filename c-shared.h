#ifndef __C_SHARED_H__
#define __C_SHARED_H__

#include "c-shared-varray.h"

#define VARNAM_SUCCESS 0
#define VARNAM_MISUSE  1
#define VARNAM_ERROR   2

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

Suggestion* makeSuggestion(char* word, int weight, int learned_on);

TransliterationResult* makeResult(varray* exact_match, varray* suggestions, varray* greedy_tokenized, int dictionary_result_count);

#endif /* __C_SHARED_H__ */