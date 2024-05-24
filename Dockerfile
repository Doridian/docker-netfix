FROM golang:1.22-alpine AS builder

ENV CGO_ENABLED=0

WORKDIR /app

COPY go.mod /app
COPY go.sum /app
RUN go mod download

COPY . /app
RUN go build -ldflags="-s -w" -trimpath -o /docker-netfix .

FROM alpine:3.20
COPY LICENSE /LICENSE

RUN apk --no-cache add util-linux
RUN mkdir -p /rootfs

COPY --from=builder --chown=0:0 --chmod=755 /docker-netfix /docker-netfix

ENTRYPOINT ["/docker-netfix", "--rootfs", "/rootfs"]
