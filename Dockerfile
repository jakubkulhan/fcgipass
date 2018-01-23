FROM golang:1.9
COPY fcgipass.go .
RUN go get -d -v github.com/tomasen/fcgi_client
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-extldflags "-static"' -o fcgipass

FROM scratch
COPY --from=0 /go/fcgipass /fcgipass
ENTRYPOINT ["/fcgipass"]
