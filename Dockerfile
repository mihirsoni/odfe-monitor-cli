FROM golang:1.14-alpine AS builder

RUN apk add --no-cache git
WORKDIR /go/src/github.com/mihirsoni/odfe-monitor-cli
COPY . .
RUN go get -v &&\
    go build -v

FROM alpine:latest

COPY --from=builder /go/src/github.com/mihirsoni/odfe-monitor-cli/odfe-monitor-cli /usr/local/bin
WORKDIR /odfe-monitor-cli

ENTRYPOINT [ "odfe-monitor-cli" ]
