PROGRAM := tavis
VERSION := 0.0.1

BASE := $(shell pwd)

MAIN_PATH := $(BASE)/cmd/tavis/main.go

BIN_DIR := $(BASE)
BIN_FILE := $(BIN_DIR)/$(PROGRAM)


BUILD_PKG := $(shell head -1 $(BASE)/go.mod | cut -d ' ' -f 2)

BUILD_DATE := $(shell date -u +%Y-%m-%d.%H:%M:%S-%Z)
GIT_COMMIT := $(shell git rev-parse HEAD)

LDFLAGS :=  -ldflags "\
	-X $(BUILD_PKG).version=$(VERSION) \
	-X $(BUILD_PKG).date=$(BUILD_DATE) \
	-X $(BUILD_PKG).commit=$(GIT_COMMIT)" \


build:
	go build $(LDFLAGS) -o $(BIN_FILE) $(MAIN_PATH)

clean:
	rm $(BIN_FILE)

.PHONY: clean

$(PROGRAM): build


all: build

