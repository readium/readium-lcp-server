#!/bin/sh

set -e

PWD=`pwd`
if [ `basename "$PWD"` = 'docker' ]
then
  echo "try cd .. && ./docker/dockerbuild.sh \`pwd\`"
  exit 1
fi

if [ `arch` = "arm64" ]
then
  echo "arm64 detected, so, set platform to linux/amd64"
  PLATFORM=--platform=linux/amd64
else
  PLATFORM=
fi

if [ -d "$1" ] 
then

  echo "run dockers"

  echo "==============="
  echo "=  LCPSERVER  ="
  echo "==============="
 # docker build -f docker/lcpserver/Dockerfile -t lcpserver:latest $PLATFORM $1

  echo "==============="
  echo "=  LSDSERVER  ="
  echo "==============="
# docker build -f docker/lsdserver/Dockerfile -t lsdserver:latest $PLATFORM $1

  echo "==============="
  echo "=  FRONTEND   ="
  echo "==============="
 # docker build -f docker/frontend/Dockerfile -t frontendtestserver:latest $PLATFORM $1

  echo "==============="
  echo "=    FINAL    ="
  echo "==============="
  docker build -f docker/Dockerfile -t lcpmasterserver:latest $PLATFORM $1

else
  echo "ERROR arg '$1' doesn't exists"
  echo "try $0 \`pwd\`"
  exit 1

fi

