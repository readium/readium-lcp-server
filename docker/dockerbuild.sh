#!/bin/sh

set -e

PWD=`pwd`
if [ `basename "$PWD"` = 'docker' ]
then
  echo "try cd .. && ./docker/dockerbuild.sh \`pwd\`"
  exit 1
fi

# if [ `arch` = "arm64" ]
# then
#   echo "arm64 detected, so, set platform to linux/amd64"
#   PLATFORM=--platform=linux/amd64
# else
#   PLATFORM=
# fi
PLATFORM=

if [ -d "$1" ] 
then

  echo "run dockers"

  if [ "$2" != "master" ]
  then
    echo "==============="
    echo "=  LCPSERVER  ="
    echo "==============="
    docker build -f docker/lcpserver/Dockerfile -t lcpserver:latest $PLATFORM $1

    echo "==============="
    echo "=  LSDSERVER  ="
    echo "==============="
    docker build -f docker/lsdserver/Dockerfile -t lsdserver:latest $PLATFORM $1

    echo "==============="
    echo "=  FRONTEND   ="
    echo "==============="
    docker build -f docker/frontend/Dockerfile -t frontendtestserver:latest $PLATFORM $1
  fi

#--no-cache
  echo "==============="
  echo "=    MASTER   ="
  echo "==============="
  docker build \
    --progress=plain\
    -f docker/Dockerfile \
    --build-arg LIBUSERKEY_PATH="./_/libuserkey.a" \
    --build-arg USERKEYH_PATH="./_/userkey.h" \
    --build-arg USERKEYGO_PATH="./_/user_key.go" \
    --build-arg BUILD_PROD=true \
    --build-arg PRIVATE_KEY_PATH="./_/privkey-edrlab.pem" \
    --build-arg CERTIFICATE_PATH="./_/cert-edrlab.pem" \
    --build-arg PROFILE="2.2" \
    -t lcpmasterserver:latest $PLATFORM $1

else
  echo "ERROR arg '$1' doesn't exists"
  echo "try $0 \`pwd\`"
  exit 1

fi

