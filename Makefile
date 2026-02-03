SHELL := /usr/bin/env bash
.DEFAULT_GOAL := help
OUT_DIR := cmd/helmper
BINARY := $(OUT_DIR)/helmper
GO_ENV ?= GOCACHE=$(CURDIR)/.gocache
.PHONY: help build test clean $(BINARY)
help:
	@echo "Available targets:"
	@echo "  make build             - build helmper with version (dev, sha) and current date -> $(BINARY)"
	@echo "  make test              - run go tests"
	@echo "  make clean             - remove build artifacts"
GIT_COMMIT := $(shell git rev-parse --short HEAD)
BUILD_DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LD_FLAGS := -ldflags="-X github.com/ChristofferNissen/helmper/internal.version=dev -X github.com/ChristofferNissen/helmper/internal.commit=$(GIT_COMMIT) -X github.com/ChristofferNissen/helmper/internal.date=$(BUILD_DATE)"
$(BINARY):
	# @mkdir -p $(dir $@)
	cd cmd/helmper && $(GO_ENV) go build $(LD_FLAGS) -o $(CURDIR)/cmd/helmper/helmper .
build: $(BINARY)
test:
	$(GO_ENV) cd cmd/helmper && go test -v ./...
clean:
	rm -rf .cache .gocache || true
	rm -f $(OUT_DIR)/helmper
