FROM golang:1.16-alpine as builder

ENV CGO_ENABLED=0
WORKDIR /go/app

RUN apk --update add ca-certificates

COPY . .

RUN go mod download

RUN go build -o /go/bin/lcpserver ./lcpserver
RUN go build -o /go/bin/lsdserver ./lsdserver
RUN go build -o /go/bin/lcpencrypt ./lcpencrypt
RUN go build -o /go/bin/frontend ./frontend

RUN mkdir -p \
  /opt/readium/db \
  /opt/readium/files/encrypted

FROM node:14-alpine as frontendbuilder

COPY --from=builder /go/app /go/app

RUN apk add --no-cache --update --virtual .build-deps python make

# Additional node modules for frontend
WORKDIR /go/app/frontend/manage
RUN yarn && yarn build

# FROM scratch
# FROM scratch as lcpserver
FROM alpine as lcpserver

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /go/bin/lcpserver /go/bin/lcpserver
COPY --from=builder /opt/readium /opt/readium

ENTRYPOINT ["/go/bin/lcpserver"]

FROM scratch as lsdserver
# FROM scratch

COPY --from=builder /go/bin/lsdserver /go/bin/lsdserver
COPY --from=builder /opt/readium /opt/readium

ENTRYPOINT ["/go/bin/lsdserver"]

FROM scratch as lcpencrypt

COPY --from=builder /go/bin/lcpencrypt /go/bin/lcpencrypt
COPY --from=builder /opt/readium /opt/readium

FROM node:14-alpine as testfrontend

COPY --from=builder /opt/readium /opt/readium
COPY --from=frontendbuilder \
  /go/app/frontend/manage \
  /go/app/frontend/manage

COPY --from=builder /go/bin/frontend /go/bin/frontend
# RUN mkdir -p /opt/readium/files/raw/frontend/uploads

ENTRYPOINT ["/go/bin/frontend"]