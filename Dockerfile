FROM golang:1.11
COPY . /src
RUN set -ex \
    && cd /src \
    && CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-extldflags "-static"' -o /bin/fcgipass

FROM scratch
COPY --from=0 /bin/fcgipass /fcgipass
ENTRYPOINT ["/fcgipass"]
