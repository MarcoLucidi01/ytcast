.POSIX:

GO = go

all: generate fmt vet test build

build: generate
	$(GO) $@

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
