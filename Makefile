BUILD_TAG = devel
ARCH ?= $(shell uname -m)
BIN := catnip
DESTDIR :=
GO ?= go
PKGNAME := catnip
PREFIX := /usr/local

MAJORVERSION := 1
MINORVERSION := 8
PATCHVERSION := 5
VERSION ?= ${MAJORVERSION}.${MINORVERSION}.${PATCHVERSION}

MAIN_DIR := ./cmd/catnip

LDFLAGS :=  -ldflags "\
	-X main.version=${VERSION} \
	-linkmode=external \
	"

SOURCES ?= $(shell find . -name "*.go" -type f)


build: $(BIN)

clean:
	rm $(BIN)

.PHONY: clean

all: build

.PHONY: install
install: build
	install -Dm755 ${BIN} $(DESTDIR)$(PREFIX)/bin/${BIN}

.PHONY: uninstall
uninstall:
	rm -f $(DESTDIR)$(PREFIX)/bin/${BIN}

$(BIN): $(SOURCES)
	$(GO) build $(FLAGS) $(LDFLAGS) -o $@ $(EXTRA_FLAGS) $(MAIN_DIR)
