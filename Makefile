.DEFAULT_GOAL := build

.PHONY:default
default: build ;

CLI_BIN := varnamcli
INSTALL_PREFIX := /usr/local
VERSION := $(shell git describe --abbrev=0 --tags | sed s/v//)
RELEASE_NAME := govarnam-${VERSION}
UNAME := $(shell uname)

SED := sed -i

ifeq ($(UNAME), Darwin)
  SED := sed -i ""
endif

pc:
	cp govarnam.pc.in govarnam.pc
	${SED} "s#@INSTALL_PREFIX@#${INSTALL_PREFIX}#g" govarnam.pc
	${SED} "s#@VERSION@#${VERSION}#g" govarnam.pc

# Used only for building the CLI
temp-pc:
	cp govarnam.pc.in govarnam.pc
	${SED} "s#@INSTALL_PREFIX@#$(realpath .)#g" govarnam.pc
	${SED} "s#@VERSION@#${VERSION}#g" govarnam.pc

	${SED} "s#/include/libgovarnam##g" govarnam.pc
	${SED} "s#/lib##g" govarnam.pc

install-script:
	cp install.sh.in install.sh
	${SED} "s#@INSTALL_PREFIX@#${INSTALL_PREFIX}#g" install.sh
	${SED} "s#@VERSION@#${VERSION}#g" install.sh
	chmod +x install.sh

install:
	./install.sh install

cli:
	go build -o ${CLI_BIN} ./cli

library-nosqlite:
	go build -tags libsqlite3 -buildmode=c-shared -o libgovarnam.so

library:
	go build -tags "fts5" -buildmode=c-shared -o libgovarnam.so

.PHONY: nix
nix:
	$(MAKE) library

	$(MAKE) temp-pc
	PKG_CONFIG_PATH=$(realpath .):$$PKG_CONFIG_PATH $(MAKE) cli

	$(MAKE) pc
	$(MAKE) install-script

.PHONY:
build:
	$(MAKE) nix

release:
	mkdir -p ${RELEASE_NAME} ${RELEASE_NAME}/schemes
	cp ${CLI_BIN} ${RELEASE_NAME}/
	cp libgovarnam.so ${RELEASE_NAME}/
	cp *.h ${RELEASE_NAME}/
	cp *.pc ${RELEASE_NAME}/
	cp install.sh ${RELEASE_NAME}/

	cp schemes/*.vst ${RELEASE_NAME}/schemes/

	zip -r ${RELEASE_NAME}.zip ${RELEASE_NAME}/*

test-govarnamgo:
	go test -count=1 govarnamgo/*.go

test:
	go test -tags fts5 -count=1 govarnam/*.go
	$(MAKE) library
	$(MAKE) test-govarnamgo
