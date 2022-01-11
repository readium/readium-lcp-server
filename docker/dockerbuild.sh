#!/bin/sh

if [ ! -d "$1" ] 
then

  echo "run dockers"

  docker build -f docker/lcpserver/Dockerfile -t lcpserver:latest $1
  docker build -f docker/lsdserver/Dockerfile -t lsdserver:latest $1
  docker build -f docker/frontend/Dockerfile -t frontendtestserver:latest $1

else

  echo "$1 doesn't exist ERROR"

fi

