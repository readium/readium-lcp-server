#!/bin/sh
set -e

READONLY_BOOL=${READONLY_BOOL=false}
AES256=${AES256=_GCM}
READIUM_LCPSERVER_CONFIG=${READIUM_LCPSERVER_CONFIG=/config.yaml}

envsubst < $1 > ${READIUM_LCPSERVER_CONFIG}

echo "exec $2"

echo "================="

exec $2
