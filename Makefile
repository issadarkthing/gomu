.PHONY: test build 

GIT_PATH=github.com/issadarkthing
BIN_NAME=gomu
REPO_NAME=gomu
BIN_DIR := $(CURDIR)/bin
INSTALL_DIR := $${HOME}/.local/bin
VERSION  := $(shell git describe --abbrev=0 --tags)
GIT_COMMIT= $(shell git rev-parse HEAD)
BUILD_DATE= $(shell date '+%Y-%m-%d-%H:%M:%S')
GO     = go

default: test build release

test:
	go test

$(BIN_DIR):
	@mkdir -p $@

$(INSTALL_DIR):
	@mkdir -p $@

build: test $(BIN_DIR) 
	${GO} build -v -o $(BIN_DIR)/$(BIN_NAME)

install: build $(INSTALL_DIR)
	cp ${BIN_DIR}/${BIN_NAME} ${INSTALL_DIR}/${BIN_NAME}

release: build
	mkdir -p dist
	tar czf dist/gomu-${VERSION}-amd64.tar.gz bin/${BIN_NAME}
