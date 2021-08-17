# Built following https://medium.com/@chemidy/create-the-smallest-and-secured-golang-docker-image-based-on-scratch-4752223b7324

# STEP 1 build executable binary
FROM golang:alpine as builder
RUN apk update && apk add --no-cache git ca-certificates && update-ca-certificates
ENV USER=appuser
ENV UID=10001

# See https://stackoverflow.com/a/55757473/12429735RUN
RUN adduser \
    --disabled-password \
    --gecos "" \
    --home "/nonexistent" \
    --shell "/sbin/nologin" \
    --no-create-home \
    --uid "${UID}" \
    "${USER}"
WORKDIR $GOPATH/src/yoyostile/spritmonitor-exporter/
COPY . .

RUN go mod init
RUN go get -d -v
RUN GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /go/bin/spritmonitor-exporter

# STEP 2 build a small image
# start from scratch
FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group
COPY --from=builder /go/bin/hello /go/bin/hello
USER appuser:appuser

EXPOSE 8086
USER appuser
ENTRYPOINT ["/go/bin/spritmonitor-exporter", "-listen-address", ":8086"]


