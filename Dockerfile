FROM --platform=${BUILDPLATFORM:-linux/amd64} golang:1.24-alpine AS builder

ARG TARGETPLATFORM
ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH

ENV CGO_ENABLED=0
ENV GOOS=${TARGETOS}
ENV GOARCH=${TARGETARCH}

WORKDIR /src
COPY go.mod go.sum /src/
RUN go mod download

COPY . /src
RUN go build -ldflags='-s -w' -trimpath -o /docker-netfix .

FROM --platform=${TARGETPLATFORM:-linux/amd64} alpine:3.21
COPY LICENSE /LICENSE

RUN apk --no-cache add util-linux
RUN mkdir -p /rootfs

COPY --from=builder --chown=0:0 --chmod=755 /docker-netfix /docker-netfix

ENTRYPOINT ["/docker-netfix", "--rootfs", "/rootfs"]
