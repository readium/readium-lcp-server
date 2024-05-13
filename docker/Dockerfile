FROM ubuntu:20.04 as box

WORKDIR /lcp
COPY . /lcp/.

ARG BUILD_PROD=false
ARG LIBUSERKEY_PATH=/docker/dummy
ARG USERKEYH_PATH=/docker/dummy
ARG USERKEYGO_PATH=/docker/dummy

COPY $LIBUSERKEY_PATH /libuserkey.a
COPY $USERKEYH_PATH /userkey.h
COPY $USERKEYGO_PATH /user_key.go

# RUN apk add build-base

RUN set -x; echo $CONTAINER_TIMEZONE && arch && uname && apt-get -y update && apt-get -y upgrade && ln -snf /usr/share/zoneinfo/$CONTAINER_TIMEZONE /etc/localtime && echo $CONTAINER_TIMEZONE > /etc/timezone && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends tzdata && apt-get install -y make curl git python build-essential bsdmainutils golang-1.21-go && ls /usr/lib/go-1.21 && export GOROOT=/usr/lib/go-1.21 && export PATH=$GOROOT/bin:$PATH && echo $PATH && which go && whereis go && go version && go env && curl -fsSL https://deb.nodesource.com/setup_20.x | bash && apt-get install -y nodejs

#go get golang.org/dl/go1.21@latest
# RUN curl -OL https://golang.org/dl/go1.21.6.linux-amd64.tar.gz && mkdir -p ./golang && tar -C ./golang -xvf go1.21.6.linux-amd64.tar.gz
ENV GOROOT=/usr/lib/go-1.21
ENV PATH=$GOROOT/bin:$PATH
RUN echo $PATH && echo ${PATH}
RUN which go && whereis go && go version && go env

# RUN CGO_ENABLED=1 GOPATH=/lcp/build GOARCH=amd64 go install -x -ldflags '-linkmode external -w -extldflags -static' ./lcpserver
RUN make BUILD_PROD=${BUILD_PROD} LIBUSERKEY_PATH=/libuserkey.a USERKEYH_PATH=/userkey.h USERKEYGO_PATH=/user_key.go lcpserver && ls -als -R /lcp/build/bin

RUN make lsdserver && ls -als -R /lcp/build/bin

RUN make frontend && ls -als -R /lcp/build/bin

RUN make "frontend/manage" && ls -als -R /lcp/build/bin && ls -als /lcp/build/frontend/manage

FROM nginx:stable-alpine

COPY --from=box /lcp/build/bin/lcpserver /usr/bin/lcpserver
COPY --from=box /lcp/build/bin/lsdserver /usr/bin/lsdserver
COPY --from=box /lcp/build/bin/frontend /usr/bin/frontend
COPY --from=box /lcp/build/frontend/manage/. /frontend/manage/.

ARG I18N=/messages
ARG AUTH_FILE=/test/htpasswd 
ARG PRIVATE_KEY_PATH=/test/cert/privkey-edrlab-test.pem
ARG CERTIFICATE_PATH=/test/cert/cert-edrlab-test.pem
ARG CONFIG_PATH=/docker/config.yaml
ARG PROFILE=basic
ARG LOGGING_SLACK_TOKEN=""
ARG LOGGING_SLACK_CHANNEL=""

COPY $PRIVATE_KEY_PATH /privkey.pem
COPY $CERTIFICATE_PATH /cert.pem
#COPY $I18N /i18n
COPY $AUTH_FILE /htpasswd

# envsubst in inittab once
COPY $CONFIG_PATH /tmp/config.yaml
COPY /docker/init.sh /init.sh

COPY /docker/inittab /etc/inittab

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

ENV PROFILE=$PROFILE

ENV PORT=8080

ENV BASE_URL=http://127.0.0.1:8080/frontend/

ENV LOGGING_SLACK_TOKEN=$LOGGING_SLACK_TOKEN
ENV LOGGING_SLACK_CHANNEL=$LOGGING_SLACK_CHANNEL

COPY /docker/nginx.conf.template /etc/nginx/templates/default.conf.template

RUN apk --update add --no-cache libc6-compat gettext sed ca-certificates openssl && update-ca-certificates && ls -als /usr/bin/lcpserver && ls -als /usr/bin/lsdserver && ls -als /usr/bin/frontend && ls -als /frontend/manage/ && ls -als /usr/bin/
#RUN ldd /usr/bin/lcpserver
#RUN ldd /usr/bin/lsdserver
#RUN ldd /usr/bin/frontend

CMD ["init"]

EXPOSE $PORT
