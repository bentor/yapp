BINARY ?= yapp
MODULE_DIR := $(CURDIR)/src
CMD_DIR := ./cmd/yapp
BIN_DIR := $(CURDIR)/bin
GOCACHE ?= $(CURDIR)/.gocache
GOMODCACHE ?= $(CURDIR)/.gomodcache
TEST_PDFS ?= $(wildcard examples/*.pdf) $(wildcard examples/*/*.pdf)
TEST_PDFS := $(strip $(TEST_PDFS))
TEST_PDFS_ABS := $(if $(TEST_PDFS),$(abspath $(TEST_PDFS)),)
TEST_ARGS ?=
ARGS ?=

.PHONY: all build run test fmt clean

all: build

build:
	@mkdir -p $(BIN_DIR)
	GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) go -C $(MODULE_DIR) build -o $(BIN_DIR)/$(BINARY) $(CMD_DIR)

run: build
	$(BIN_DIR)/$(BINARY) $(ARGS)

test:
	@echo "TEST_PDFS=$(TEST_PDFS_ABS)"
	TEST_PDFS="$(TEST_PDFS_ABS)" GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) go -C $(MODULE_DIR) test ./... $(TEST_ARGS)

fmt:
	GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) go -C $(MODULE_DIR) fmt ./...

clean:
	rm -rf $(BIN_DIR)
