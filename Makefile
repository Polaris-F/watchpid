SHELL := /bin/bash

MODULE := github.com/Polaris-F/watchpid
BINARY := watchpid
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w \
	-X $(MODULE)/internal/buildinfo.Version=$(VERSION) \
	-X $(MODULE)/internal/buildinfo.Commit=$(COMMIT) \
	-X $(MODULE)/internal/buildinfo.Date=$(DATE)

.PHONY: test build build-all release clean

test:
	go test ./...

build:
	mkdir -p bin
	go build -trimpath -ldflags "$(LDFLAGS)" -o bin/$(BINARY) ./cmd/watchpid

build-all:
	VERSION=$(VERSION) COMMIT=$(COMMIT) DATE=$(DATE) ./scripts/release.sh $(VERSION)

release: build-all

clean:
	rm -rf bin dist
