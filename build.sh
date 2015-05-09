#!/bin/bash

echo "Using gopath: $GOPATH"
deps=(
    "golang.org/x/tools/cmd/vet|master"
    "github.com/golang/lint/golint|master"
    "golang.org/x/tools/cmd/cover|master"
    "github.com/chihaya/bencode|master"
    "github.com/garyburd/redigo/redis|master"
    "github.com/kisielk/raven-go/raven|master"
    "github.com/labstack/echo|v0.0.12"
    "github.com/julienschmidt/httprouter|master"
    "github.com/Sirupsen/logrus|master"
    "github.com/goji/param|master"
    "github.com/influxdb/influxdb/client|master"
    "github.com/goji/httpauth|master"
)
pushd &> /dev/null
for dep in "${deps[@]}"
do
    repo=$(echo ${dep} | cut -f1 -d\|)
    version=$(echo ${dep} | cut -f2 -d\|)

    echo "Cloning $repo..."
    go get ${repo}

    pushd ${GOPATH}/src/${repo} &> /dev/null
        echo "Checking out $repo @ $version"
        git fetch
        git pull
        git checkout ${version}
    popd &> /dev/null
done

make && go vet && make test
