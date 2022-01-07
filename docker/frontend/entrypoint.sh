#!/bin/bash
set -e

RIGHT_COPY_NUMBER=${RIGHT_COPY_NUMBER=10}
RIGHT_PRINT_NUMBER=${RIGHT_PRINT_NUMBER=2000}

READIUM_FRONTEND_CONFIG=/build/config.yaml

envsubst < $1 > ${READIUM_FRONTEND_CONFIG}


# Update base href of frontend index.html
sed -i "s@base href=\"/\"@base href=\"${BASE_URL}\"@g" /src/github.com/readium/readium-lcp-server/frontend/manage/index.html

exec "$2"
