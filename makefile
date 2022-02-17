.POSIX:
.PHONY: all build test fmt vet cross-build major-release minor-release patch-release install uninstall clean

VERSION  = $(shell git describe --tags)
GO       = go
GOFLAGS  = -trimpath -tags netgo,osusergo -ldflags="-w -s -X main.progVersion=$(VERSION)"
RELEASE  = ./release
PREFIX   = /usr/local
PROGNAME = ytcast
CROSSTARGETS = linux-386 linux-amd64 linux-arm linux-arm64 darwin-amd64 darwin-arm64

all: fmt vet test build

build:
	$(GO) build $(GOFLAGS) -o $(PROGNAME)

test:
	$(GO) test ./...

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

cross-build: all
	for target in $(CROSSTARGETS); do \
		env $$(echo "$$target" | awk -F '-' '{ print "GOOS="$$1, "GOARCH="$$2 }') \
			$(GO) build $(GOFLAGS) -o "$(PROGNAME)-$(VERSION)-$$target"; \
	done

major-release: all
	$(RELEASE) major

minor-release: all
	$(RELEASE) minor

patch-release: all
	$(RELEASE) patch

install: all
	mkdir -p $(DESTDIR)$(PREFIX)/bin
	install -m 755 $(PROGNAME) $(DESTDIR)$(PREFIX)/bin

uninstall:
	rm $(DESTDIR)$(PREFIX)/bin/$(PROGNAME)

clean:
	$(GO) clean ./...
	rm -rf *.tmp $(PROGNAME)-v*
