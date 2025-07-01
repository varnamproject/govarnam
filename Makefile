.PHONY: all install uninstall
.PHONY: test-govarnamgo test
.PHONY: library-nosqlite library-mac-universal release

UNAME = $(shell uname)
TOP = $(realpath .)

DESTDIR ?=
PREFIX ?= /usr/local
BINDIR ?= $(PREFIX)/bin
INCLUDEDIR ?= $(PREFIX)/include
LIBDIR ?= $(PREFIX)/lib
PKGCONFIGDIR ?= $(PREFIX)/lib/pkgconfig

GO = go
LIPO = lipo
ZIP = zip
INSTALL = install

HEADERS = libgovarnam.h \
	c-shared.h \
	c-shared-util.h \
	c-shared-varray.h

PKGCONFIG_IN = govarnam.pc.in
PKGCONFIG = $(PKGCONFIG_IN:.in= )

CLI_BIN = varnamcli
CLI_DIR = $(TOP)/cli

VERSION = 1.9.1
RELEASE_NAME = govarnam-$(VERSION)-$(shell uname -m)

LINKERNAME = libgovarnam.so
SONAME = $(LINKERNAME).$(shell (echo $(VERSION) | cut -d. -f1))
LIBNAME = $(LINKERNAME).$(VERSION)

ifeq ($(UNAME), Darwin)
	LIBNAME = libgovarnam.dylib
else
	EXT_LDFLAGS = -extldflags -Wl,-soname,$(SONAME),--version-script,$(TOP)/govarnam.syms
endif

VERSION_STAMP_LDFLAGS = -X 'github.com/varnamproject/govarnam/govarnam.VersionString=$(VERSION)' $(EXT_LDFLAGS)

all: $(LIBNAME) $(CLI_BIN)

$(LIBNAME):
	CGO_ENABLED=1 $(GO) build -tags "fts5" \
		-buildmode=c-shared \
		-ldflags "-s -w $(VERSION_STAMP_LDFLAGS)" \
		-o $@ .
	mv libgovarnam.so.*.h libgovarnam.h

$(CLI_BIN): $(LIBNAME)
	sed -e "s|@PREFIX@|$(TOP)|g" \
		-e "s|@INCLUDEDIR@/libgovarnam|$(TOP)|g" \
		-e "s|@LIBDIR@|$(TOP)|g" \
		-e "s|@VERSION@|$(VERSION)|g" \
		$(PKGCONFIG_IN) > $(PKGCONFIG)
	ln -sf $(LIBNAME) $(LINKERNAME)
	ln -sf $(LIBNAME) $(SONAME)
	export PKG_CONFIG_PATH=$(TOP):$$PKG_CONFIG_PATH \
	&& export LD_LIBRARY_PATH=$(TOP):$$LD_LIBRARY_PATH \
	&& $(GO) build -o $@ -ldflags "-s -w" $(CLI_DIR)

install: $(PKGCONFIG)
	mkdir -p $(DESTDIR)$(BINDIR)
	mkdir -p $(DESTDIR)$(INCLUDEDIR)/libgovarnam
	mkdir -p $(DESTDIR)$(LIBDIR)
	mkdir -p $(DESTDIR)$(PKGCONFIGDIR)
	$(INSTALL) -Dpm 0755 $(CLI_BIN) $(DESTDIR)$(BINDIR)
	for i in $(HEADERS) ;\
		do $(INSTALL) -Dpm 0644 $$i \
			$(DESTDIR)$(INCLUDEDIR)/libgovarnam ;\
	done
	cd $(DESTDIR)$(LIBDIR) \
		&& ln -sf $(LIBNAME) $(SONAME) \
		&& ln -sf $(LIBNAME) $(LINKERNAME)
	$(INSTALL) -Dpm 0644 $(LIBNAME) $(DESTDIR)$(LIBDIR)
	$(INSTALL) -Dpm 0644 $(PKGCONFIG) $(DESTDIR)$(PKGCONFIGDIR)

$(PKGCONFIG): $(PKGCONFIG_IN) $(CLI_BIN)
	sed -e "s|@PREFIX@|$(PREFIX)|g" \
		-e "s|@INCLUDEDIR@|$(INCLUDEDIR)|g" \
		-e "s|@LIBDIR@|$(LIBDIR)|g" \
		-e "s|@VERSION@|$(VERSION)|g" \
		$< > $@

uninstall:
	rm $(DESTDIR)$(BINDIR)/$(CLI_BIN)
	for i in $(HEADERS) ;\
		do rm $(DESTDIR)$(INCLUDEDIR)/libgovarnam/$$i ;\
	done
	rm -r $(DESTDIR)$(INCLUDEDIR)/libgovarnam
	for i in $(LIBNAME) $(SONAME) $(LINKERNAME) ;\
		do rm $(DESTDIR)$(LIBDIR)/$$i ;\
	done
	rm $(DESTDIR)$(PKGCONFIGDIR)/$(PKGCONFIG)

clean:
	rm -f $(CLI_BIN) \
		libgovarnam.h \
		$(LIBNAME) \
		$(SONAME) \
		$(LINKERNAME) \
		$(PKGCONFIG)

library-nosqlite:
	CGO_ENABLED=1 $(GO) build -tags "fts5,libsqlite3" \
		-buildmode=c-shared \
		-ldflags "-s -w $(VERSION_STAMP_LDFLAGS)" \
		-o $(LIBNAME) .

library-mac-universal:
	GOOS=darwin GOARCH=arm64 $(MAKE) $(LIBNAME)
	mv $(LIBNAME) $(LIBNAME).arm64
	GOOS=darwin GOARCH=amd64 $(MAKE) $(LIBNAME)
	mv $(LIBNAME) $(LIBNAME).amd64
	$(LIPO) -create -output $(LIBNAME) \
		$(LIB_NAME).arm64 \
		$(LIB_NAME).amd64

release:
	echo "Hope you have updated version in constants.go"
	mkdir -p $(RELEASE_NAME)
	cp ${CLI_BIN} ${RELEASE_NAME}
	cp libgovarnam.so* $(RELEASE_NAME)
	cp *.h $(RELEASE_NAME)
	cp *.pc $(RELEASE_NAME)
	$(ZIP) -r $(RELEASE_NAME).zip $(RELEASE_NAME)/*

test-govarnamgo:
	export PKG_CONFIG_PATH=$(TOP):$$PKG_CONFIG_PATH \
	&& export LD_LIBRARY_PATH=$(TOP):$$LD_LIBRARY_PATH \
	&& govarnamgo/run_tests.sh

test:
	$(GO) test -tags fts5 -count=1 -cover govarnam/*.go
	$(MAKE) all
	$(MAKE) test-govarnamgo
