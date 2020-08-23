.PHONY: test build format install release

GIT_PATH     = github.com/issadarkthing
BIN_NAME     = gomu
REPO_NAME    = gomu
BIN_DIR     := $(CURDIR)/bin
INSTALL_DIR := $${HOME}/.local/bin
VERSION      = $(shell git describe --abbrev=0 --tags)
GIT_COMMIT   = $(shell git rev-parse HEAD)
BUILD_DATE   = $(shell date '+%Y-%m-%d-%H:%M:%S')
GO           = go

default: build

run: format test build
	$(BIN_DIR)/$(BIN_NAME)

test:
	@echo === TESTING ===
	go test

format:
	@echo === FORMATTING ===
	go fmt *.go

$(BIN_DIR):
	mkdir -p $@

$(INSTALL_DIR):
	mkdir -p $@

build: $(BIN_DIR) 
	@echo === BUILDING ===
	${GO} build -v -o $(BIN_DIR)/$(BIN_NAME)

install: build $(INSTALL_DIR)
	@echo === INSTALLING ===
	cp ${BIN_DIR}/${BIN_NAME} ${INSTALL_DIR}/${BIN_NAME}

release: build
	@echo === RELEASING ===
	mkdir -p dist
	tar czf dist/gomu-${VERSION}-amd64.tar.gz bin/${BIN_NAME}
