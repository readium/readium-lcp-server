#!/bin/bash

echo "Shutting down and removing Docker containers..."

# Stop and remove Docker containers; also removes network
docker-compose down

echo "Removing Docker volumes..."

# Get current directory name, which is a prefix of the Docker volume names
DIR_PATH=$(dirname "$(readlink -f "$0")")
PREFIX=$(basename "$DIR_PATH")
DBDATA=$PREFIX"_dbdata"
ENCFILES=$PREFIX"_encfiles"
RAWFILES=$PREFIX"_rawfiles"
# Remove volumes
docker volume rm $DBDATA
docker volume rm $ENCFILES
docker volume rm $RAWFILES

echo "Deleting downloaded or built Docker images..."

# Remove images
docker rmi readium/testfrontend:working
docker rmi readium/lsdserver:working
docker rmi readium/lcpserver:working
docker rmi readium/lcpencrypt:working
docker rmi atmoz/sftp:alpine
docker rmi mariadb:latest

echo "Resetting config file"
# Load env values to test which config file to restore ()
. .env
# Reset config.yaml file to initial version with ENV refs (with ./etc bind-mount, 
# ENV refs were changed to hard-coded values)
if [ -z "$AWS_S3_USER" ]; then
  cp etc/config.yaml.fs etc/config.yaml
else
  cp etc/config.yaml.s3 etc/config.yaml
fi

echo "Resources removed successfully."
