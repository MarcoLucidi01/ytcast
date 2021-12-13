.POSIX:

GO = go
VERSION = $(shell git describe --tags)

all: generate fmt vet test build

run: generate
	$(GO) $@ .

build: generate
	$(GO) $@ -ldflags="-X main.progVersion=$(VERSION)"

test: generate
	$(GO) $@ ./...

generate:
	$(GO) $@ ./...

fmt:
	$(GO) $@ ./...

vet:
	$(GO) $@ ./...

clean:
	$(GO) $@ ./...
