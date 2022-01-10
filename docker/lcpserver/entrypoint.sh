#!/bin/sh
set -e

READIUM_LCPSERVER_CONFIG=/config.yaml

envsubst < $1 > ${READIUM_LCPSERVER_CONFIG}

echo "exec $2"

echo "================="

exec $2
