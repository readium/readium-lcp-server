
FROM node:6.9.2 as manage-builder
WORKDIR /
RUN make frontend/manage
COPY /build/frontend/manage/. /build/frontend/manage/.

FROM go:latest as frontend-builder
WORKDIR /
RUN make frontend
COPY /build/bin/frontend /build/bin/frontend

FROM alpine:latest
WORKDIR /
COPY /docker/frontend/run.sh run.sh
COPY /docker/frontend/config.yaml /tmp/config.yaml
COPY --from=manage-builder /build/frontend/manage/. /build/frontend/manage/.
COPY --from=fronten-builder /build/bin/frontend /build/bin/frontend
EXPOSE ${PORT}
CMD sh run.sh /tmp/config.yaml
