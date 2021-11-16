.POSIX:

GO = go

all: generate fmt vet test build

run: generate
	$(GO) $@ .

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
