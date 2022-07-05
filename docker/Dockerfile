FROM lcpserver:latest as lcpserver
FROM lsdserver:latest as lsdserver
FROM frontendtestserver:latest as frontendtestserver

FROM nginx:stable-alpine

COPY --from=lcpserver /bin/lcpserver /lcpserver/lcpserver
COPY /docker/lcpserver/entrypoint.sh /lcpserver/entrypoint.sh
COPY /docker/lcpserver/config.yaml /lcpserver/tmp/config.yaml

ENV READIUM_LCPSERVER_CONFIG=/lcpserver/config.yaml

COPY --from=lsdserver /bin/lsdserver /lsdserver/lsdserver
COPY /docker/lsdserver/entrypoint.sh /lsdserver/entrypoint.sh
COPY /docker/lsdserver/config.yaml /lsdserver/tmp/config.yaml

ENV READIUM_LSDSERVER_CONFIG=/lsdserver/config.yaml

COPY --from=frontendtestserver /bin/frontend /frontend/frontend
COPY --from=frontendtestserver /manage/. /frontend/manage/.
COPY /docker/frontend/entrypoint.sh /frontend/entrypoint.sh
COPY /docker/frontend/config.yaml /frontend/tmp/config.yaml

COPY /test/cert/cert-edrlab-test.pem /cert.pem
COPY /test/cert/privkey-edrlab-test.pem /privkey.pem
COPY /messages /i18n
COPY /test/htpasswd /htpasswd

COPY /docker/inittab /etc/inittab

RUN apk add gettext

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

COPY /docker/nginx.conf.template /etc/nginx/templates/

CMD ["init"]

EXPOSE ${PORT}
