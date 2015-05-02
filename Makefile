.PHONY: all test clean build install
# export GOPATH=`pwd`
GOFLAGS = -ldflags "-X main.version `git rev-parse --short HEAD`"

all: build


build:
	@go build $(GOFLAGS)

install:
	@go get $(GOFLAGS) ./...

test: install
	@go test $(GOFLAGS)

cover: install
	@go test -coverprofile=`pwd`/coverage.out $(GOFLAGS) ./... && go tool cover -html=`pwd`/coverage.out

bench: install
	@go test -run=NONE -bench=. $(GOFLAGS) ./...

clean:
	@go clean $(GOFLAGS) -i ./...

## EOF