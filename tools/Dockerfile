FROM golang:1.17.1-alpine AS deps
RUN apk --no-cache add upx pkgconfig opus-dev opusfile-dev opus opus-tools build-base zip make patchelf
WORKDIR /project
COPY go.mod ./
COPY go.sum ./
RUN go mod download

FROM deps AS builder
ENV GOOS linux
ENV GOARCH amd64
ENV CGO_ENABLED 0
CMD ["make", "build"]

