#! /bin/bash

PORT=8080

docker kill mylcp
docker rm mylcp

./docker/dockerbuild.sh `pwd` master

docker run \
  --name mylcp \
  --publish $PORT:$PORT \
  --env PORT=$PORT \
  --volume /tmp/lcp:/lcp \
  --platform linux/amd64 \
  lcpmasterserver
