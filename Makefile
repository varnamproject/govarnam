.DEFAULT_GOAL := build

.PHONY:default
default: build ;

CLI_BIN := varnamcli
INSTALL_PREFIX := $(or ${PREFIX},${PREFIX},/usr/local)

# Try to get the commit hash from git
LAST_COMMIT := $(or $(shell git rev-parse --short HEAD 2> /dev/null),"UNKNOWN")
VERSION := $(or $(shell echo $$(git describe --abbrev=0 --tags || echo "latest") | sed s/v//),"v0.0.0")
BUILDSTR := ${VERSION} (\#${LAST_COMMIT} $(shell date -u +"%Y-%m-%dT%H:%M:%S%z"))

RELEASE_NAME := govarnam-${VERSION}-${shell uname -m}
UNAME := $(shell uname)

SED := sed -i
LIB_NAME := libgovarnam.so
SO_NAME := $(shell (echo $(VERSION) | cut -d. -f1))
CURDIR := $(shell pwd)

ifeq ($(UNAME), Darwin)
  SED := sed -i ""
  LIB_NAME = libgovarnam.dylib
else
  EXT_LDFLAGS = -extldflags "-Wl,-soname,$(LIB_NAME).$(SO_NAME),--version-script,$(CURDIR)/govarnam.syms"
endif

VERSION_STAMP_LDFLAGS := -X 'github.com/varnamproject/govarnam/govarnam.BuildString=${BUILDSTR}' -X 'github.com/varnamproject/govarnam/govarnam.VersionString=${VERSION}' $(EXT_LDFLAGS)
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
	${SED} "s#/lib\$$##g" govarnam.pc

install.sh: install.sh.in
	cp install.sh.in install.sh
	${SED} "s#@INSTALL_PREFIX@#${INSTALL_PREFIX}#g" install.sh
	${SED} "s#@VERSION@#${VERSION}#g" install.sh
	${SED} "s#@LIB_NAME@#${LIB_NAME}#g" install.sh
	${SED} "s#@SO_NAME@#${SO_NAME}#g" install.sh
	chmod +x install.sh

.PHONY: install
install: install.sh
	./install.sh install

.PHONY: uninstall
uninstall: install.sh
	./install.sh uninstall



.PHONY: cli
cli:
	go build -o ${CLI_BIN} -ldflags "-s -w" ./cli

library-nosqlite:
	CGO_ENABLED=1 go build -tags "fts5,libsqlite3" -buildmode=c-shared -ldflags "-s -w ${VERSION_STAMP_LDFLAGS}" -o ${LIB_NAME} .

library:
	CGO_ENABLED=1 go build -tags "fts5" -buildmode=c-shared -ldflags "-s -w ${VERSION_STAMP_LDFLAGS}" -o ${LIB_NAME} .
	ln -sf "$(realpath ./)/libgovarnam.so" "$(realpath ./)/libgovarnam.so.${SO_NAME}"

library-mac-universal:
	GOOS=darwin GOARCH=arm64 $(MAKE) library
	mv ${LIB_NAME} ${LIB_NAME}.arm64
	GOOS=darwin GOARCH=amd64 $(MAKE) library
	mv ${LIB_NAME} ${LIB_NAME}.amd64
	lipo -create -output ${LIB_NAME} ${LIB_NAME}.arm64 ${LIB_NAME}.amd64

.PHONY: nix
nix:
	$(MAKE) library

	$(MAKE) temp-pc
	PKG_CONFIG_PATH=$(realpath .):$$PKG_CONFIG_PATH $(MAKE) cli

	$(MAKE) pc
	$(MAKE) install.sh

.PHONY:
build:
	$(MAKE) nix

release:
	echo "Hope you have updated version in constants.go"
	mkdir -p ${RELEASE_NAME}
	cp ${CLI_BIN} ${RELEASE_NAME}/
	cp libgovarnam.so ${RELEASE_NAME}/
	cp *.h ${RELEASE_NAME}/
	cp *.pc ${RELEASE_NAME}/
	cp install.sh ${RELEASE_NAME}/

	zip -r ${RELEASE_NAME}.zip ${RELEASE_NAME}/*

test-govarnamgo:
	$(MAKE) temp-pc
	PKG_CONFIG_PATH=$(realpath .):$$PKG_CONFIG_PATH LD_LIBRARY_PATH=$(realpath .):$$LD_LIBRARY_PATH govarnamgo/run_tests.sh

test:
	go test -tags fts5 -count=1 -cover govarnam/*.go

	$(MAKE) library
	$(MAKE) test-govarnamgo

.PHONY: clean
clean:
	rm -f varnamcli libgovarnam.*  govarnam.pc install.sh
