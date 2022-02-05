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

TransliterationResult* makeResult(varray* exact_words, varray* exact_matches, varray* dictionary_suggestions, varray* pattern_dictionary_suggestions, varray* tokenizer_suggestions, varray* greedy_tokenized)
{
  TransliterationResult *result = (TransliterationResult*) malloc (sizeof(TransliterationResult));
  result->ExactWords = exact_words;
  result->ExactMatches = exact_matches;
  result->DictionarySuggestions = dictionary_suggestions;
  result->PatternDictionarySuggestions = pattern_dictionary_suggestions;
  result->TokenizerSuggestions = tokenizer_suggestions;
  result->GreedyTokenized = greedy_tokenized;

  return result;
}

void destroySuggestions(void* pointer)
{
  if (pointer != NULL) {
    Suggestion* sug = (Suggestion*) pointer;
    free(sug->Word);
    sug->Word = NULL;
    free(sug);
    sug = NULL;
  }
}

void destroySuggestionsArray(varray* pointer)
{
  varray_free(pointer, &destroySuggestions);
  pointer = NULL;
}

void destroyTransliterationResult(TransliterationResult* result)
{
  destroySuggestionsArray(result->ExactMatches);
  destroySuggestionsArray(result->DictionarySuggestions);
  destroySuggestionsArray(result->PatternDictionarySuggestions);
  destroySuggestionsArray(result->TokenizerSuggestions);
  destroySuggestionsArray(result->GreedyTokenized);
  result->ExactMatches = NULL;
  result->DictionarySuggestions = NULL;
  result->PatternDictionarySuggestions = NULL;
  result->TokenizerSuggestions = NULL;
  result->GreedyTokenized = NULL;
  free(result);
  result = NULL;
}

SchemeDetails* makeSchemeDetails(char* Identifier, char* LangCode, char* DisplayName, char* Author, char* CompiledDate, bool IsStable)
{
  SchemeDetails* sd = (SchemeDetails*) malloc (sizeof(SchemeDetails));
  sd->Identifier = Identifier;
  sd->LangCode = LangCode;
  sd->DisplayName = DisplayName;
  sd->Author = Author;
  sd->CompiledDate = CompiledDate;
  sd->IsStable = IsStable;

  return sd;
}

void destroySchemeDetails(void* pointer)
{
  if (pointer != NULL) {
    Suggestion* sug = (Suggestion*) pointer;
    free(sug->Word);
    sug->Word = NULL;
    free(sug);
    sug = NULL;
  }
}

void destroySchemeDetailsArray(void* cSchemeDetails)
{
  varray_free(cSchemeDetails, &destroySchemeDetails);
}

LearnStatus makeLearnStatus(int TotalWords, int FailedWords)
{
  LearnStatus ls;
  ls.TotalWords = TotalWords;
  ls.FailedWords = FailedWords;
  return ls;
}

Symbol* makeSymbol(int Identifier, int Type, int MatchType, char* Pattern, char* Value1, char* Value2, char* Value3, char* Tag, int Weight, int Priority, int AcceptCondition, int Flags)
{
  Symbol *symbol = (Symbol*) malloc (sizeof(Symbol));
  symbol->Identifier = Identifier;
  symbol->Type = Type;
  symbol->MatchType = MatchType;
  symbol->Pattern = Pattern;
  symbol->Value1 = Value1;
  symbol->Value2 = Value2;
  symbol->Value3 = Value3;
  symbol->Tag = Tag;
  symbol->Weight = Weight;
  symbol->Priority = Priority;
  symbol->AcceptCondition = AcceptCondition;
  symbol->Flags = Flags;
  return symbol;
}

void destroySymbol(void* pointer)
{
  if (pointer != NULL) {
    Symbol* symbol = (Symbol*) pointer;
    free(symbol->Pattern),
    free(symbol->Value1);
    free(symbol->Value2);
    free(symbol->Value3);
    free(symbol->Tag);
    symbol->Pattern = NULL;
    symbol->Value1 = NULL;
    symbol->Value2 = NULL;
    symbol->Value3 = NULL;
    symbol->Tag = NULL;
    free(symbol);
    symbol = NULL;
  }
}

void destroySymbolArray(void* cSymbols)
{
  varray_free(cSymbols, &destroySymbol);
}
