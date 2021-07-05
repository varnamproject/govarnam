#ifndef __VARRAY_H__
#define __VARRAY_H__

#include "c-shared-util.h"

/**
 * Array to hold pointers. This expands automatically.
 *
 **/
typedef struct varray_t
{
    void **memory;
    size_t allocated;
    size_t used;
    int index;
} varray;

VARNAM_EXPORT extern varray* 
varray_init();

VARNAM_EXPORT extern void
varray_push(varray *array, void *data);

VARNAM_EXPORT extern int
varray_length(varray *array);

VARNAM_EXPORT extern bool
varray_is_empty (varray *array);

VARNAM_EXPORT extern bool
varray_exists (varray *array, void *item, bool (*equals)(void *left, void *right));

VARNAM_EXPORT extern void
varray_clear(varray *array);

VARNAM_EXPORT extern void*
varray_get(varray *array, int index); 

VARNAM_EXPORT extern void
varray_insert(varray *array, int index, void *data);

VARNAM_EXPORT extern void
varray_free(varray *array, void (*destructor)(void*));

#endif /* VARRAY_H */