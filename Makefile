.PHONY: all test clean build install
# export GOPATH=`pwd`
GOFLAGS = -ldflags "-X mika.Version=`git rev-parse --short HEAD`"

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
	@go run $(GOFLAGS) -race cmd/mika/mika.go

install: deps
	@go install $(GOFLAGS) ./...

test:
	@go test $(GOFLAGS) -cover . ./...

cover:
	@go test -coverprofile=coverage.out $(GOFLAGS) */*_test.go
	@go tool cover -html=coverage.out

bench:
	@go test -run=NONE -bench=. $(GOFLAGS) ./...

clean:
	@go clean $(GOFLAGS) -i
	@rm mika

## EOF