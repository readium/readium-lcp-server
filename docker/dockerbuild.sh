#!/bin/sh

set -e

if [ `arch` = "arm64" ]
then
  echo "arm64 detected set platform linux/amd64"
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
  docker build -f docker/lcpserver/Dockerfile -t lcpserver:latest $PLATFORM $1

  echo "==============="
  echo "=  LSDSERVER  ="
  echo "==============="
  docker build -f docker/lsdserver/Dockerfile -t lsdserver:latest $PLATFORM $1

  echo "==============="
  echo "=  FRONTEND   ="
  echo "==============="
  docker build -f docker/frontend/Dockerfile -t frontendtestserver:latest $PLATFORM $1

else

  echo "$1 doesn't exists ERROR"

fi

