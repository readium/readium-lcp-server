#!/bin/sh


set -e

mkdir /etc/lcp

git clone https://github.com/readium/readium-lcp-server.git /etc/lcp

cd /etc/lcp

git checkout cd

echo "$PWD"
ls -la


docker run --rm -v /var/run/docker.sock:/var/run/docker.sock -v $PWD:$PWD -w=$PWD docker/compose:1.29.2 -f $PWD/docker/docker-compose.yml up

