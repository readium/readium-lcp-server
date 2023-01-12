#! /bin/bash

set -x # expands variable and prints line

PORT=${1:-8080}
VOLUME_NAME=${2:-/tmp/lcp}
NAME=${3:-mylcp}
CONTAINER_NAME=${4:-lcpmasterserver}

docker kill $NAME
docker rm $NAME

# uncomment this line to build
# ./docker/dockerbuild.sh `pwd` master

  # --platform linux/amd64 \
docker run \
  --name $NAME \
  --publish $PORT:$PORT \
  --env PORT=$PORT \
  --volume $VOLUME_NAME:/lcp \
  $CONTAINER_NAME
