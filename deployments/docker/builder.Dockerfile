ARG GOLANG_VERSION=1.17-buster
ARG CGO=0
ARG OS=linux
ARG ARCH=amd64

FROM golang:${GOLANG_VERSION} AS gobuilder

ARG CGO
ARG OS
ARG ARCH

RUN go version
WORKDIR /go/src
COPY . .
RUN cd cmd/simplem2m && \
    CGO_ENABLED='${CGO}' GOOS='${OS}' GOARCH='${ARCH}' \
    make -f /go/src/Makefile build
