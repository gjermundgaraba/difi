GO ?= go
GOLANGCI_LINT ?= golangci-lint
BIN_DIR ?= bin
BINARY ?= difi

.PHONY: build install-binary lint lint-fix test tidy check-all

build:
	mkdir -p $(BIN_DIR)
	$(GO) build -o $(BIN_DIR)/$(BINARY) ./cmd/difi

install-binary:
	$(GO) install ./cmd/difi

lint:
	$(GOLANGCI_LINT) run

lint-fix:
	$(GOLANGCI_LINT) run --fix

test:
	$(GO) test ./...

tidy:
	$(GO) mod tidy

check-all: tidy build test lint
