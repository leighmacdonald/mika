#!/bin/bash
export GOPATH=`pwd`
export GOBIN=$GOPATH/bin
echo "Using gopath: $GOPATH"
go get -v golang.org/x/tools/cmd/vet
go get -v github.com/golang/lint/golint
go get -v golang.org/x/tools/cmd/cover
go get -v github.com/chihaya/bencode
go get -v github.com/garyburd/redigo/redis
go get -v github.com/kisielk/raven-go/raven
go get -v github.com/labstack/echo
go get -v github.com/julienschmidt/httprouter
go get -v github.com/goji/param
go get -v github.com/influxdb/influxdb/client
go get -v github.com/goji/httpauth
mkdir -p src/git.totdev.in/totv
ln -s `pwd` src/git.totdev.in/totv/mika
make && go vet && make test