FROM golang:latest

ENV READIUM_LCPSERVER_CONFIG config.yaml

RUN go get -v github.com/endigo/readium-lcp-server/lcpserver

EXPOSE 8990

CMD ["lcpserver"]