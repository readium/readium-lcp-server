#!/bin/sh
set -e

READIUM_LSDSERVER_CONFIG=/config.yaml

READONLY_BOOL=${READONLY_BOOL=false}

envsubst < $1 > ${READIUM_LSDSERVER_CONFIG}

echo "exec $2"

echo "================="

exec $2
