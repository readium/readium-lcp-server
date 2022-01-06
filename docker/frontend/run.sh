
envsubst < $1 > /build/config.yaml

READIUM_FRONTEND_CONFIG=/build/config.yaml
.$2
