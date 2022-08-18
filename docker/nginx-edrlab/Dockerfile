
FROM nginx:stable-alpine

ARG SSL_PRIVATE_KEY_PATH=/
ARG SSL_CERTIFICATE_PATH=/

COPY $SSL_PRIVATE_KEY_PATH /ssl.key
COPY $SSL_CERTIFICATE_PATH /ssl.crt

COPY /docker/nginx-edrlab/nginx.conf.template /etc/nginx/templates/

EXPOSE 443
