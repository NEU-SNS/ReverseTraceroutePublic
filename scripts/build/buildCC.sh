#!/bin/bash

# build and save the docker image for the central controller

set -e

BIN_DIR=./bin
BIN=controller
CONT_DIR=containers
ROOT=$(git rev-parse --show-toplevel)

cp $BIN_DIR/$BIN cmd/$BIN/docker/.
cd cmd/$BIN/docker

docker build --rm=true -t revtr/controller .
docker save -o $ROOT/$CONT_DIR/cc.tar revtr/controller
# docker rmi revtr/controller
