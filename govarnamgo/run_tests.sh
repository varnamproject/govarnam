#!/bin/bash

tmpDir=$(mktemp -d -t govarnamgo-test-XXXXXX)

# We do this bash scripting way to set environment dir because
# Go makes the environment while a C library loads, hence dynamic
# environment variable setting won't work
# https://github.com/golang/go/wiki/cgo#environmental-variables

export VARNAM_LEARNINGS_DIR=$tmpDir
go test -count=1 -cover govarnamgo/*.go

rm -rf $tmpDir