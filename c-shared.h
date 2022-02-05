#ifndef __C_SHARED_H__
#define __C_SHARED_H__

#include "c-shared-varray.h"

#define VARNAM_SUCCESS 0
#define VARNAM_MISUSE  1
#define VARNAM_ERROR   2
#define VARNAM_CANCELLED  3

#define VARNAM_CONFIG_USE_DEAD_CONSONANTS 100
#define VARNAM_CONFIG_IGNORE_DUPLICATE_TOKEN 101
// VARNAM_CONFIG_ENABLE_SUGGESTIONS hasn't been implemented yet 
#define VARNAM_CONFIG_ENABLE_SUGGESTIONS 102
#define VARNAM_CONFIG_USE_INDIC_DIGITS 103
#define VARNAM_CONFIG_SET_DICTIONARY_SUGGESTIONS_LIMIT 104
#define VARNAM_CONFIG_SET_PATTERN_DICTIONARY_SUGGESTIONS_LIMIT 105
#define VARNAM_CONFIG_SET_TOKENIZER_SUGGESTIONS_LIMIT 106
#define VARNAM_CONFIG_SET_DICTIONARY_MATCH_EXACT 107

typedef struct Suggestion_t {
  char* Word;
  int Weight;
  int LearnedOn;
} Suggestion;

typedef struct TransliterationResult_t {
  varray* ExactWords;
  varray* ExactMatches;
  varray* DictionarySuggestions;
  varray* PatternDictionarySuggestions;
  varray* TokenizerSuggestions;
  varray* GreedyTokenized;
} TransliterationResult;

Suggestion* makeSuggestion(char* word, int weight, int learned_on);

TransliterationResult* makeResult(varray* exact_words, varray* exact_matches, varray* dictionary_suggestions, varray* pattern_dictionary_suggestions, varray* tokenizer_suggestions, varray* greedy_tokenized);

void destroySuggestionsArray(varray* pointer);
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

typedef struct LearnStatus_t {
  int TotalWords;
  int FailedWords;
} LearnStatus;

LearnStatus makeLearnStatus(int TotalWords, int FailedWords);

typedef struct Symbol_t {
  int Identifier;
  int Type;
  int MatchType;
  char* Pattern;
  char* Value1;
  char* Value2;
  char* Value3;
  char* Tag;
  int Weight;
  int Priority;
  int AcceptCondition;
  int Flags;
} Symbol;

Symbol* makeSymbol(int Identifier, int Type, int MatchType, char* Pattern, char* Value1, char* Value2, char* Value3, char* Tag, int Weight, int Priority, int AcceptCondition, int Flags);

void destroySymbolArray(void* cSymbols);

#endif /* __C_SHARED_H__ */