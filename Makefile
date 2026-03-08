.PHONY: build install clean test

BINARY_NAME=ttg
INSTALL_PATH=$(HOME)/.local/bin/$(BINARY_NAME)
GO=go
GOFLAGS=-v

build:
	$(GO) build $(GOFLAGS) -o $(BINARY_NAME) ./cmd

install: build
	install -d $(HOME)/.local/bin
	install -m 755 $(BINARY_NAME) $(INSTALL_PATH)

clean:
	rm -f $(BINARY_NAME)

test:
	$(GO) test -v ./...

run:
	$(GO) run ./cmd

fmt:
	$(GO) fmt ./...

lint:
	golangci-lint run ./... || go vet ./...

.DEFAULT_GOAL := build
