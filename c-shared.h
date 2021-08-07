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
  varray* ExactMatches;
  varray* DictionarySuggestions;
  varray* PatternDictionarySuggestions;
  varray* TokenizerSuggestions;
  varray* GreedyTokenized;
} TransliterationResult;

Suggestion* makeSuggestion(char* word, int weight, int learned_on);

TransliterationResult* makeResult(varray* exact_matches, varray* dictionary_suggestions, varray* pattern_dictionary_suggestions, varray* tokenizer_suggestions, varray* greedy_tokenized);

void destroyTransliterationResult(TransliterationResult*);

typedef struct SchemeDetails_t {
  char* Identifier;
  char* LangCode;
  char* DisplayName;
  char* Author;
  char* CompiledDate;
  bool IsStable;
} SchemeDetails;

SchemeDetails* makeSchemeDetails(char* Identifier, char* LangCode, char* DisplayName, char* Author, char* CompiledDate, bool IsStable);

void destroySchemeDetailsArray(void* cSchemeDetails);

#endif /* __C_SHARED_H__ */