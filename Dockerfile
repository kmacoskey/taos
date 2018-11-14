FROM golang:1.11-alpine as builder

WORKDIR /go/src/github.com/kmacoskey/taos

COPY . .

RUN apk --no-cache add git make && \
    go get ./... && \
    make build

FROM alpine:latest

COPY --from=builder /go/src/github.com/kmacoskey/taos/taos .

CMD ["./taos"]
