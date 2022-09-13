#!/bin/bash

# Should be run from [...]/ReverseTraceroute
# build and save the docker image for the atlas
# the cert and conf should already be in the docker dir

set -e

BIN_DIR=./bin
BIN=atlas

cp $BIN_DIR/$BIN cmd/$BIN/docker/.


cd cmd/$BIN/docker
# Copy the rankingservice code to run RIPE traceroutes and atlas rr intersection here, TODO should be written in go.

rsync -a  --exclude '*resources*' --include '*.py'  ~/go/src/github.com/NEU-SNS/ReverseTraceroute/rankingservice .
# Some resources are needed 
mkdir -p rankingservice/resources/
mkdir -p rankingservice/atlas/resources/refresh/
cp ~/go/src/github.com/NEU-SNS/ReverseTraceroute/rankingservice/resources/ripe_probes_atlas.json rankingservice/resources/

CONT_DIR=containers
ROOT=$(git rev-parse --show-toplevel)

docker build --rm=true -t revtr/atlas .
docker save -o $ROOT/$CONT_DIR/atlas.tar revtr/atlas
# docker rmi revtr/atlas
