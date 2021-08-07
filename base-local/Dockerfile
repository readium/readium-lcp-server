FROM golang:1.14-alpine3.11 AS builder

RUN apk add --no-cache --update --virtual .build-deps \
  git \
  build-base \
  nodejs \
  npm \
  gettext

# Create Go binaries
RUN go get -v github.com/endigo/readium-lcp-server/...

# RUN go build -o /go/bin/lcpserver lcpserver
# RUN go build -o /go/bin/lsdserver lsdserver
# RUN go build -o /go/bin/lcpencrypt lcpencrypt
# RUN go build -o /go/bin/frontend frontend

# Additional node modules for frontend
# WORKDIR /go/src/github.com/endigo/readium-lcp-server/frontend/manage
# RUN npm install && npm run-script build

RUN mkdir -p \
  /opt/readium/db \
  /opt/readium/files/encrypted

###############################################################################
## FINAL BASE IMAGE
###############################################################################

FROM alpine:3.11 AS base

RUN apk add --no-cache --update --virtual .runtime-deps \
  libintl

COPY --from=builder /usr/bin/envsubst /usr/bin/envsubst
COPY --from=builder /opt/readium /opt/readium
COPY etc /etc/readium

ENV READIUM_LCPSERVER_CONFIG /etc/readium/config.yaml
ENV READIUM_PUBLIC_HOST 0.0.0.0
ENV READIUM_PRIVATE_HOST 127.0.0.1

###############################################################################
## FINAL ENCRYPT
###############################################################################

# This (mixing Python and Go in one codebase) is not ideal, but I'm a Python programmer,
# not a Go programmer...  :p

FROM python:3.8-alpine3.11 AS lcpencrypt

RUN apk add --no-cache --update --virtual .runtime-deps \
  libintl \
  && pip install watchdog[watchmedo]

COPY --from=builder /usr/bin/envsubst /usr/bin/envsubst
COPY --from=builder /opt/readium /opt/readium
COPY etc /etc/readium
COPY --from=builder /go/bin/lcpencrypt /go/bin/lcpencrypt
RUN mkdir -p /opt/readium/files/raw && cp /etc/readium/watcher.yaml /opt/readium/files/raw/watcher.yaml

WORKDIR "/opt/readium/files/encrypted"
ENTRYPOINT ["watchmedo", "tricks-from", "/opt/readium/files/raw/watcher.yaml"]

###############################################################################
## FINAL LCP SERVER
###############################################################################

FROM scratch AS lcpserver

COPY --from=builder /go/bin/lcpserver /go/bin/lcpserver
# COPY files/aws_credentials /root/.aws/credentials
ENV READIUM_LCPSERVER_CONFIG=/etc/readium/config.yaml
# CMD ["sh", "-c", "envsubst < /etc/readium/config.yaml > /tmp/config.yaml && mv /tmp/config.yaml /etc/readium/config.yaml; /go/bin/lcpserver"]
ENTRYPOINT [ "/go/bin/lcpserver" ]

###############################################################################
## FINAL LSD SERVER
###############################################################################

FROM scratch AS lsdserver

COPY --from=builder /go/bin/lsdserver /go/bin/lsdserver
ENV READIUM_LSDSERVER_CONFIG=/etc/readium/config.yaml
# CMD ["sh", "-c", "envsubst < /etc/readium/config.yaml > /tmp/config.yaml && mv /tmp/config.yaml /etc/readium/config.yaml; /go/bin/lsdserver"]
ENTRYPOINT [ "/go/bin/lsdserver" ]

###############################################################################
## FINAL TEST FRONTEND SERVER
###############################################################################

FROM base AS testfrontend

COPY --from=builder \
  /go/src/github.com/endigo/readium-lcp-server/frontend/manage \
  /go/src/github.com/endigo/readium-lcp-server/frontend/manage
COPY --from=builder /go/bin/frontend /go/bin/frontend
RUN mkdir -p /opt/readium/files/raw/frontend/uploads

ENV READIUM_FRONTEND_CONFIG=/etc/readium/config.yaml
CMD ["sh", "-c", "mkdir -p /opt/readium/files/raw/frontend/uploads; envsubst < /etc/readium/config.yaml > /tmp/config.yaml && mv /tmp/config.yaml /etc/readium/config.yaml; /go/bin/frontend"]
