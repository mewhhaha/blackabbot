FROM golang:1.17.1-alpine AS deps
RUN apk --no-cache add upx pkgconfig opus-dev opusfile-dev build-base
WORKDIR /build
COPY go.mod ./
COPY go.sum ./
RUN go mod download

FROM deps AS builder
ARG CMD_NAME
COPY . ./
RUN go build -ldflags="-s -w" -o run ./cmd/$CMD_NAME
RUN upx ./run

FROM alpine
RUN apk --no-cache add pkgconfig opus-dev opusfile-dev
COPY --from=builder /build/run /bin/run
ENTRYPOINT [ "run" ]
