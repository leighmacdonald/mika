.PHONY: all test clean build install

GOFLAGS = -ldflags "-X main.version `git rev-parse --short HEAD`" -race

all: build


build:
	@go build $(GOFLAGS)

install:
	@go get $(GOFLAGS) ./...

test: install
	@go test $(GOFLAGS) ./...

bench: install
	@go test -run=NONE -bench=. $(GOFLAGS) ./...

clean:
	@go clean $(GOFLAGS) -i ./...

## EOF