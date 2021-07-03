/* Dynamically growing array implementation
 *
 * Copyright (C) Navaneeth.K.N
 *
 * This is part of libvarnam licensed under MPL-2.0
 */



#include <stdio.h>
#include <stdlib.h>
#include <assert.h>
#include "c-shared-varray.h"
#include "c-shared-util.h"

varray*
varray_init()
{
    varray *array = (varray*) malloc (sizeof(varray));
    array->memory = NULL;
    array->allocated = 0;
    array->used = 0;
    array->index = -1;

    return array;
} 

void
varray_push(varray *array, void *data)
{
    size_t toallocate;
    size_t size = sizeof(void*);

    if (data == NULL) return;

    if ((array->allocated - array->used) < size) {
        toallocate = array->allocated == 0 ? size : (array->allocated * 2);
        array->memory = realloc(array->memory, toallocate);
        array->allocated = toallocate;
    }

    array->memory[++array->index] = data;
    array->used = array->used + size;
}

int
varray_length(varray *array)
{
    if (array == NULL)
        return 0;

    return array->index + 1;
}

bool
varray_is_empty (varray *array)
{
    return (varray_length (array) == 0);
}

void
varray_clear(varray *array)
{
    int i;
    for(i = 0; i < varray_length(array); i++)
    {
        array->memory[i] = NULL;
    }
    array->used = 0;
    array->index = -1;
}

void
varray_free(varray *array, void (*destructor)(void*))
{
    int i;
    void *item;

    if (array == NULL)
        return;

    if (destructor != NULL)
    {
        for(i = 0; i < varray_length(array); i++)
        {
            item = varray_get (array, i);
            if (item != NULL) destructor(item);
        }
    }

    if (array->memory != NULL)
        free(array->memory);
    free(array);
}

void*
varray_get(varray *array, int index)
{
    if (index < 0 || index > array->index)
        return NULL;

    assert(array->memory);

    return array->memory[index];
}

void
varray_insert(varray *array, int index, void *data)
{
    if (index < 0 || index > array->index)
        return;

    array->memory[index] = data;
}

bool
varray_exists (varray *array, void *item, bool (*equals)(void *left, void *right))
{
    int i;

    for (i = 0; i < varray_length (array); i++)
    {
        if (equals(varray_get (array, i), item))
            return true;
    }

    return false;
}
