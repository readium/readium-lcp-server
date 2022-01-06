
envsubst < $1 > /build/config.yaml

READIUM_FRONTEND_CONFIG=/build/config.yaml
./build/bin/frontend
