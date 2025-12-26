BINARY ?= yapp
SRC_DIR := ./src
BIN_DIR := ./bin
TEST_PDFS ?= $(wildcard examples/*.pdf) $(wildcard examples/*/*.pdf)
TEST_ARGS ?=

.PHONY: all build run test fmt clean

all: build

build:
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/$(BINARY) $(SRC_DIR)

run: build
	$(BIN_DIR)/$(BINARY) $(ARGS)

test:
	@echo "TEST_PDFS=$(TEST_PDFS)"
	TEST_PDFS="$(TEST_PDFS)" go test ./... $(TEST_ARGS)

fmt:
	go fmt ./...

clean:
	rm -rf $(BIN_DIR)
