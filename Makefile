.PHONY: all test clean build install
# export GOPATH=`pwd`
GOFLAGS = -ldflags "-X git.totdev.in/totv/mika.Version=`git rev-parse --short HEAD`"

all: build

deps:
	@./update.sh

vet:
	@go vet . ./...

fmt:
	@go fmt . ./...

build: fmt
	@go build $(GOFLAGS) cmd/mika/mika.go

run:
	@go run $(GOFLAGS) cmd/mika/mika.go

install: deps
	@go install $(GOFLAGS) ./...

test:
	@go test $(GOFLAGS) -cover . ./...

cover: install
	@go test -coverprofile=coverage.out $(GOFLAGS) */*_test.go
	@go tool cover -html=coverage.out

bench: install
	@go test -run=NONE -bench=. $(GOFLAGS) . ./...

clean:
	@go clean $(GOFLAGS) -i
	@rm mika

## EOF