#!/bin/bash

# build and save the docker image for the revtr service

set -e 

BIN_DIR=./bin
BIN=revtr
CONT_DIR=containers
ROOT=$(git rev-parse --show-toplevel)


cp $BIN_DIR/$BIN cmd/$BIN/docker/.
cp -r cmd/$BIN/certs cmd/$BIN/docker/.
cp -r cmd/revtr/ip2as cmd/revtr/docker/
cp -r $ROOT/cmd/revtr/webroot $ROOT/cmd/revtr/docker

cd cmd/$BIN/docker


docker build --rm=true -t revtr/revtr .
docker save -o $ROOT/$CONT_DIR/revtr.tar revtr/revtr
# docker rmi revtr/revtr

# rm -rf $ROOT/cmd/revtr/docker/webroot
