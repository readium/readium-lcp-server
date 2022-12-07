#! /bin/bash

PORT=8080

docker kill mylcp
docker rm mylcp

# uncomment this line to build
# ./docker/dockerbuild.sh `pwd` master

  # --platform linux/amd64 \
docker run \
  --name mylcp \
  --publish $PORT:$PORT \
  --env PORT=$PORT \
  --volume /tmp/lcp:/lcp \
  lcpmasterserver
