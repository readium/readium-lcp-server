
FROM golang:alpine as lcpserver-builder
WORKDIR /lcp
COPY . /lcp/.

RUN apk add build-base
RUN make lcpserver



FROM alpine:latest
WORKDIR /

RUN apk add --no-cache gettext # envsubst

COPY /docker/lcpserver/entrypoint.sh /entrypoint.sh
COPY /docker/lcpserver/config.yaml /tmp/config.yaml

COPY --from=lcpserver-builder /lcp/build/bin/lcpserver /bin/lcpserver

EXPOSE ${PORT}

CMD ["/bin/lcpserver"]
ENTRYPOINT ["/entrypoint.sh", "/tmp/config.yaml"]

