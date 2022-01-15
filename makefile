.POSIX:
.PHONY: all build test fmt vet major-release minor-release patch-release install uninstall clean

VERSION = $(shell git describe --tags)
GO      = go
GOFLAGS = -ldflags="-X main.progVersion=$(VERSION)"
RELEASE = ./release
PREFIX  = /usr/local
BINARY  = ytcast

all: fmt vet test build

build:
	$(GO) build -o $(BINARY) $(GOFLAGS)

test:
	$(GO) test ./...

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

major-release: all
	$(RELEASE) major

minor-release: all
	$(RELEASE) minor

patch-release: all
	$(RELEASE) patch

install: all
	mkdir -p $(DESTDIR)$(PREFIX)/bin
	install -m 755 $(BINARY) $(DESTDIR)$(PREFIX)/bin

uninstall:
	rm $(DESTDIR)$(PREFIX)/bin/$(BINARY)

clean:
	rm -rf *.tmp
	$(GO) clean ./...
