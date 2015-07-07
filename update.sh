#!/bin/bash

echo "Using gopath: $GOPATH"

deps=(
    "golang.org/x/tools/cmd/godoc|master"
    "golang.org/x/tools/cmd/vet|master"
    "github.com/golang/lint/golint|master"
    "golang.org/x/tools/cmd/cover|master"
    "github.com/chihaya/bencode|master"
    "github.com/garyburd/redigo/redis|master"
    "github.com/julienschmidt/httprouter|master"
    "github.com/rcrowley/go-metrics|master"
    "github.com/Sirupsen/logrus|master"
    "github.com/goji/param|master"
    "github.com/gin-gonic/gin|master"
    "github.com/getsentry/raven-go|master"q
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
        git checkout ${version}
        git pull origin ${version}
    popd &> /dev/null
done
