FROM lcpserver:latest as lcpserver
FROM lsdserver:latest as lsdserver
FROM frontendtestserver:latest as frontendtestserver

FROM nginx:stable-alpine

COPY --from=lcpserver /bin/lcpserver /usr/bin/lcpserver
COPY --from=lsdserver /bin/lsdserver /usr/bin/lsdserver
COPY --from=frontendtestserver /bin/frontend /usr/bin/frontend
COPY --from=frontendtestserver /manage/. /frontend/manage/.

COPY /test/cert/cert-edrlab-test.pem /cert.pem
COPY /test/cert/privkey-edrlab-test.pem /privkey.pem
COPY /messages /i18n
COPY /test/htpasswd /htpasswd

# envsubst in inittab once
COPY /docker/config.yaml /tmp/config.yaml
COPY /docker/init.sh /init.sh

COPY /docker/inittab /etc/inittab

RUN apk add gettext sed

ENV LCP_PORT=8081
ENV LCP_HOST=127.0.0.1
ENV LSD_PORT=8082
ENV LSD_HOST=127.0.0.1
ENV FRONTEND_PORT=8083
ENV FRONTEND_HOST=127.0.0.1
ENV AUTH_FILE=/htpasswd
ENV CERTIFICATE_PATH=/cert.pem
ENV PRIVATE_KEY=/privkey.pem
ENV LOCALIZATION_FOLDER=/i18n
ENV LOCALIZATION_DEFAULT_LANGUAGE=en-US
ENV LOCALIZATION_LANGUAGES_ARRAY=[\"en-US\"]

ENV LCP_DATABASE_ROOT_PATH=/lcp/db

ENV LSD_BASE_URL=http://127.0.0.1:8080/lsdserver
ENV LSD_NOTIFY_AUTH_USER=adm_username
ENV LSD_NOTIFY_AUTH_PASS=adm_password
ENV LSD_DATABASE=sqlite3://file:$LCP_DATABASE_ROOT_PATH/lcp.sqlite?cache=shared&mode=rwc

ENV LCP_BASE_URL=http://127.0.0.1:8080/lcpserver
ENV LCP_UPDATE_AUTH_USER=adm_username
ENV LCP_UPDATE_AUTH_PASS=adm_password
ENV LCP_DATABASE=sqlite3://file:$LCP_DATABASE_ROOT_PATH/lsd.sqlite?cache=shared&mode=rwc
ENV LCP_STORAGE_PATH=/lcp/files/storage

ENV FRONTEND_BASE_URL=http://127.0.0.1:8080/frontend
ENV FRONTEND_MANAGE=/frontend/manage
ENV FRONTEND_DATABASE=sqlite3://file:$LCP_DATABASE_ROOT_PATH/frontend.sqlite?cache=shared&mode=rwc

ENV FRONTEND_MASTER_PATH=/lcp/files/master
ENV FRONTEND_ENCRYPTED_PATH=/lcp/files/encrypted
ENV FRONTEND_PROVIDER_URI=https://www.myprovidername.org

ENV READIUM_LCPSERVER_CONFIG=/config.yaml
ENV READIUM_LSDSERVER_CONFIG=/config.yaml
ENV READIUM_FRONTEND_CONFIG=/config.yaml

ENV BASE_URL=http://127.0.0.1:8080/frontend/

COPY /docker/nginx.conf.template /etc/nginx/templates/

CMD ["init"]

EXPOSE ${PORT}
