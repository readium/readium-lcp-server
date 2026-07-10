
echo "START ENV SUBSTITUTION" && envsubst < /tmp/config.yaml > /config.yaml && echo "END ENV SUBSTITUTION"

# Update base href of frontend index.html
sed -i "s@base href=\"/\"@base href=\"${BASE_URL}\"@g" $1/index.html

mkdir -p $LCP_DATABASE_ROOT_PATH
mkdir -p $LCP_STORAGE_PATH
mkdir -p $FRONTEND_MASTER_PATH
mkdir -p $FRONTEND_ENCRYPTED_PATH
