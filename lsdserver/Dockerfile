FROM golang:latest

ENV READIUM_LSDSERVER_CONFIG config.yaml

RUN go get -v github.com/endigo/readium-lcp-server/lsdserver

EXPOSE 8990

CMD ["lsdserver"]