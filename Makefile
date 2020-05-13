.PHONY: all test clean build install
GO_FLAGS = -ldflags "-X 'github.com/leighmacdonald/mika/consts.BuildVersion=`git rev-parse --short HEAD`'"

all: build

deps:
	@./update.sh

vet:
	@go vet . ./...

fmt:
	@go fmt . ./...

build: fmt
	@go build $(GO_FLAGS)

run:
	@go run $(GO_FLAGS) -race cmd/mika/mika.go

install: deps
	@go install $(GO_FLAGS) ./...

test:
	@go test $(GO_FLAGS) -cover . ./...

cover:
	@go test -coverprofile=coverage.out $(GO_FLAGS) */*_test.go
	@go tool cover -html=coverage.out

bench:
	@go test -run=NONE -bench=. $(GO_FLAGS) ./...

clean:
	@go clean $(GO_FLAGS) -i

## EOF