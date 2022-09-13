#!/bin/bash

# Build and save the docker image for the plcontroller

set -e 

BIN_DIR=./bin
BIN=plcontroller
CONT_DIR=containers
ROOT=$(git rev-parse --show-toplevel)

cp $BIN_DIR/$BIN cmd/$BIN/docker/.
cd cmd/$BIN/docker

docker build --rm=true -t revtr/plcontroller_test .
docker save -o $ROOT/$CONT_DIR/plc.tar revtr/plcontroller_test
# docker rmi revtr/plcontroller
