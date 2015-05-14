#!/usr/bin/env bash
# Build script used for gitlab
export GOPATH=`pwd`
git submodule update --init
mkdir -p src/git.totdev.in/totv
pushd src/git.totdev.in/totv
ln -s ${GOPATH} mika
popd
cp config.json.dist config.json
./build.sh -u