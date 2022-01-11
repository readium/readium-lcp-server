
FROM golang:bullseye as lsdserver-builder
WORKDIR /lcp
COPY . /lcp/.

RUN apk add build-base
RUN make lsdserver


FROM alpine:latest
WORKDIR /

RUN apk add --no-cache gettext # envsubst

COPY /docker/lsdserver/entrypoint.sh /entrypoint.sh
COPY /docker/lsdserver/config.yaml /tmp/config.yaml

COPY --from=lsdserver-builder /lcp/build/bin/lsdserver /bin/lsdserver

EXPOSE ${PORT}

CMD ["/bin/lsdserver"]
ENTRYPOINT ["/entrypoint.sh", "/tmp/config.yaml"]

