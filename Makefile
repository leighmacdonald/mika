.PHONY: all test clean build install
# export GOPATH=`pwd`
GOFLAGS = -ldflags "-X git.totdev.in/totv/mika.Version `git rev-parse --short HEAD`"

all: build


build:
	@go fmt ./...
	@go build $(GOFLAGS) cmd/mika/mika.go

install:
	@go get $(GOFLAGS) ./...

test: install
	@go test $(GOFLAGS)

cover: install
	@go test -coverprofile=`pwd`/coverage.out $(GOFLAGS) ./... && go tool cover -html=`pwd`/coverage.out

bench: install
	@go test -run=NONE -bench=. $(GOFLAGS) ./...

clean:
	@go clean $(GOFLAGS) -i
	@rm mika

## EOF