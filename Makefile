.PHONY: all test clean build install
GIT_TAG =
GO_FLAGS = -ldflags "-X 'github.com/leighmacdonald/mika/consts.BuildVersion=`git describe --abbrev=0`'"
DEBUG_FLAGS = -gcflags "all=-N -l"
# .RECIPEPREFIX = >
all: build

vet:
	@go vet . ./...

fmt:
	@go fmt . ./...

build_debug:
	@go build $(DEBUG_FLAGS) $(GO_FLAGS) -o mika

build: fmt
	@go build $(GO_FLAGS)

run:
	@go run $(GO_FLAGS) -race cmd/mika/mika.go

install: deps
	@go install $(GO_FLAGS) ./...

test:
	@go test $(GO_FLAGS) -race -cover . ./...

testcover:
	@go test -race -coverprofile c.out $(GO_FLAGS) ./...

lint:
	@golangci-lint run

bench:
	@go test -run=NONE -bench=. $(GO_FLAGS) ./...

clean:
	@go clean $(GO_FLAGS) -i

image_latest:
	@docker build -t leighmacdonald/mika:latest .

image_tag:
	docker build -t leighmacdonald/mika:$$(git describe --abbrev=0 --tags) .

docker_run: image_latest
	@docker-compose run --rm mika

## EOF