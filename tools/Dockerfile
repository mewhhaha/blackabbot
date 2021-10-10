FROM golang:1.17.1-alpine AS deps
RUN apk --no-cache add upx pkgconfig opus-dev opusfile-dev opus opus-tools build-base zip make
WORKDIR /project
COPY go.mod ./
COPY go.sum ./
RUN go mod download

FROM deps AS builder
CMD ["make", "build"]

