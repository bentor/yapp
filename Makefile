BINARY ?= yapp
CMD_DIR := ./cmd/yapp
BIN_DIR := ./bin
GOCACHE ?= ./.gocache
TEST_PDFS ?= $(wildcard examples/*.pdf) $(wildcard examples/*/*.pdf)
TEST_ARGS ?=
ARGS ?=

.PHONY: all build run test fmt clean

all: build

build:
	@mkdir -p $(BIN_DIR)
	GOCACHE=$(GOCACHE) go build -o $(BIN_DIR)/$(BINARY) $(CMD_DIR)

run: build
	$(BIN_DIR)/$(BINARY) $(ARGS)

test:
	@echo "TEST_PDFS=$(TEST_PDFS)"
	TEST_PDFS="$(TEST_PDFS)" GOCACHE=$(GOCACHE) go test ./... $(TEST_ARGS)

fmt:
	go fmt ./...

clean:
	rm -rf $(BIN_DIR)
