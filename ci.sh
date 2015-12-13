#!/usr/bin/env bash
# Build script used for gitlab
export GOPATH=`pwd`
git submodule update --init
mkdir -p src/github.com/leighmacdonald
pushd src/github.com/leighmacdonald
ln -s ${GOPATH} mika
popd
cp config.json.dist config.json
./update.sh
make