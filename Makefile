# Binary targets
BINARY_CLI = bin/cistern
BINARY_API = bin/api

.PHONY: all build test clean help

all: build

build:
	go build -o $(BINARY_CLI) ./cmd/cistern
	go build -o $(BINARY_API) ./cmd/api

test:
	go test -v ./...

clean:
	rm -rf bin

help:
	@echo "Usage:"
	@echo "  make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  build           Build all binaries (placed in bin/)"
	@echo "  test            Run all tests"
	@echo "  clean           Remove build artifacts"
