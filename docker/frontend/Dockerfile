
# FROM node:18.12.1 doesn't works npm failed to download librairies
FROM ubuntu:20.04 as manage-builder

WORKDIR /lcp
COPY . /lcp/.

RUN apt-get update \
    && apt-get install -y make curl git \
    && curl -O https://nodejs.org/dist/v18.12.1/node-v18.12.1-linux-arm64.tar.gz \
    && tar -xzf node-v18.12.1-linux-arm64.tar.gz

RUN PATH=/lcp/node-v18.12.1-linux-arm64/bin:$PATH make "frontend/manage"


FROM golang:alpine as frontend-builder
WORKDIR /lcp
COPY . /lcp/.

RUN apk add build-base
RUN make frontend


FROM alpine:latest
WORKDIR /

RUN apk add --no-cache gettext # envsubst

COPY /docker/frontend/entrypoint.sh /entrypoint.sh
COPY /docker/frontend/config.yaml /tmp/config.yaml

COPY --from=manage-builder /lcp/build/frontend/manage/. /manage/.
ENV DIRECTORY=/manage

COPY --from=frontend-builder /lcp/build/bin/frontend /bin/frontend

EXPOSE ${PORT}

CMD ["/bin/frontend"]
ENTRYPOINT ["/entrypoint.sh", "/tmp/config.yaml", "/manage"]
