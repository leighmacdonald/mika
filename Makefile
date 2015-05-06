.PHONY: all test clean build install
# export GOPATH=`pwd`
GOFLAGS = -ldflags "-X git.totdev.in/totv/mika.Version `git rev-parse --short HEAD`"

all: build

deps:
	@go get -v github.com/chihaya/bencode
	@go get -v github.com/garyburd/redigo/redis
	@go get -v github.com/kisielk/raven-go/raven
	@go get -v github.com/labstack/echo
	@go get -v github.com/influxdb/influxdb/client
	@go get -v github.com/goji/httpauth

build:
	@go fmt ./...
	@go build $(GOFLAGS) cmd/mika/mika.go

install:
	@go get $(GOFLAGS) ./...

test:
	@go test $(GOFLAGS) -cover ./util ./tracker ./cmd/mika ./conf ./db ./stats ./

cover: install
	@go test -coverprofile=coverage.out $(GOFLAGS) */*_test.go && go tool cover -html=coverage.out

bench: install
	@go test -run=NONE -bench=. $(GOFLAGS) ./...

clean:
	@go clean $(GOFLAGS) -i
	@rm mika

## EOF