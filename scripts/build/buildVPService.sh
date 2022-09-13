#!/bin/bash

# build and save the docker image for the vpservice

set -e 

BIN_DIR=./bin
BIN=vpservice
CONT_DIR=containers
ROOT=$(git rev-parse --show-toplevel)


cp $BIN_DIR/$BIN cmd/$BIN/docker/.
cd cmd/$BIN/docker

docker build --no-cache --rm=true -t revtr/vpservice .
docker save -o $ROOT/$CONT_DIR/vps.tar revtr/vpservice
# docker rmi revtr/vpservice
