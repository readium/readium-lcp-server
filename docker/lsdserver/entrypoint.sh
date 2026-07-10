#!/bin/sh
set -e

READONLY_BOOL=${READONLY_BOOL=false}
READIUM_LSDSERVER_CONFIG=${READIUM_LSDSERVER_CONFIG=/config.yaml}

envsubst < $1 > ${READIUM_LSDSERVER_CONFIG}

echo "exec $2"

echo "================="

exec $2
