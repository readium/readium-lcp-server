FROM golang:1.16-alpine as builder

ENV CGO_ENABLED=0
WORKDIR /go/app

COPY . .

RUN go mod download

RUN go build -o /go/bin/lcpserver ./lcpserver
RUN go build -o /go/bin/lsdserver ./lsdserver
RUN go build -o /go/bin/lcpencrypt ./lcpencrypt
RUN go build -o /go/bin/frontend ./frontend

RUN mkdir -p \
  /opt/readium/db \
  /opt/readium/files/encrypted

FROM node:15-alpine as frontendbuilder

COPY --from=builder /go/app /go/app

# Additional node modules for frontend
WORKDIR /go/app/frontend/manage
RUN yarn && yarn build

# FROM scratch
FROM scratch as lcpserver

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

FROM scratch as testfrontend

COPY --from=builder /opt/readium /opt/readium
COPY --from=frontendbuilder \
  /go/src/github.com/endigo/readium-lcp-server/frontend/manage \
  /go/src/github.com/endigo/readium-lcp-server/frontend/manage

COPY --from=builder /go/bin/frontend /go/bin/frontend
RUN mkdir -p /opt/readium/files/raw/frontend/uploads

ENTRYPOINT ["/go/bin/frontend"]