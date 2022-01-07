
FROM ubuntu:20.04 as manage-builder

WORKDIR /lcp
COPY . /lcp/.

RUN apt-get update \
    && apt-get install make \
    && curl -O https://nodejs.org/dist/v6.9.2/node-v6.9.2-linux-x64.tar.gz \
    && tar -xzf node-v6.9.2-linux-x64.tar.gz

RUN PATH=/lcp/node-v6.9.2-linux-x64/bin:$PATH make "frontend/manage"


FROM golang:bullseye as frontend-builder
WORKDIR /lcp
COPY . /lcp/.
RUN make frontend
# COPY /build/bin/frontend /build/bin/frontend




FROM alpine:latest
WORKDIR /

COPY /docker/frontend/run.sh /run.sh
COPY /docker/frontend/config.yaml /tmp/config.yaml

COPY --from=manage-builder /lcp/build/frontend/manage/. /manage/.
ENV DIRECTORY=/manage

COPY --from=frontend-builder /lcp/build/bin/frontend /frontend

EXPOSE ${PORT}

CMD sh /run.sh /tmp/config.yaml /frontend
