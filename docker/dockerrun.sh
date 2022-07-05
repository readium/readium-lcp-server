#! /bin/bash

LCP_BASE_URL="http://127.0.0.1:8091"
LSD_BASE_URL="http://127.0.0.1:8092"
FRONTEND_BASE_URL="http://127.0.0.1:8093"

DB_URI="mysql://admin:admin@tcp(db:3306)/readium-lcp"
MASTER_REPOSITORY="/raw"
ENCRYPTED_REPOSITORY="/encrypted"

# The usernames and passwords must match the ones in the htpasswd files for each server.
LSD_NOTIFY_AUTH_USER="adm_username"
LSD_NOTIFY_AUTH_PASS="adm_password"
LCP_UPDATE_AUTH_USER="adm_username"
LCP_UPDATE_AUTH_PASS="adm_password"

S3_ENDPOINTS="http://minio:8082"
S3_ACCESS_ID="user" #key
S3_SECRET="12345678"
S3_DISABLE_SSL_BOOL="true"
S3_BUCKET="readium-lcp"
S3_REGION=""
S3_TOKEN=""

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
