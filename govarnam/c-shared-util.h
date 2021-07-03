#ifndef __UTIL_H__
#define __UTIL_H__

#include <stddef.h>

#if defined (_WIN32)
  #if defined(varnam_EXPORTS)
    #define VARNAM_EXPORT __declspec(dllexport)
  #else
    #define VARNAM_EXPORT __declspec(dllimport)
  #endif /* varnam_EXPORTS */
#else /* defined (_WIN32) */
 #define VARNAM_EXPORT
#endif

#ifndef __cplusplus
  typedef int bool;
  #define false 0
  #define true  1
#endif

#endif /* __UTIL_H__ */