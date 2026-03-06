GO ?= go
GOLANGCI_LINT ?= golangci-lint
BIN_DIR ?= bin
BINARY ?= difi

.PHONY: build lint lint-fix test

build:
	mkdir -p $(BIN_DIR)
	$(GO) build -o $(BIN_DIR)/$(BINARY) ./cmd/difi

lint:
	$(GOLANGCI_LINT) run

lint-fix:
	$(GOLANGCI_LINT) run --fix

test:
	$(GO) test ./...
