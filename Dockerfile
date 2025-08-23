FROM golang:1.22 AS builder

WORKDIR /usr/local/src

RUN apt update && apt install -y build-essential libsqlite3-dev

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=linux go build -o /usr/local/bin/lcpencrypt ./lcpencrypt
RUN CGO_ENABLED=1 GOOS=linux go build -o /usr/local/bin/lcpserver ./lcpserver
RUN CGO_ENABLED=1 GOOS=linux go build -o /usr/local/bin/lsdserver ./lsdserver

FROM debian:12-slim

RUN apt update && apt install -y sqlite3 supervisor

RUN useradd -m readium
RUN useradd -m -g readium lcp
RUN useradd -m -g readium lsd

COPY --from=builder --chown=:readium --chmod=0550 /usr/local/bin/lcpencrypt /usr/local/bin/lcpencrypt
COPY --from=builder --chown=lcp:readium --chmod=0540 /usr/local/bin/lcpserver /usr/local/bin/lcpserver
COPY --from=builder --chown=lsd:readium --chmod=0540 /usr/local/bin/lsdserver /usr/local/bin/lsdserver
COPY --from=builder --chown=readium:readium --chmod=0640 /usr/local/src/test/cert /usr/local/var/readium/lcp/cert

COPY --chown=readium:readium --chmod=0440 docker/config.yaml /usr/local/etc/readium/config.yaml

COPY docker/supervisord.conf /etc/supervisord.conf

ENV READIUM_LCPSERVER_CONFIG=/usr/local/etc/readium/config.yaml
ENV READIUM_LSDSERVER_CONFIG=/usr/local/etc/readium/config.yaml

EXPOSE 8990

ENTRYPOINT ["supervisord", "--nodaemon", "--configuration", "/etc/supervisord.conf"]
