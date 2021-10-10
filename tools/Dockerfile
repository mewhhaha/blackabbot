FROM golang:1.17.1-alpine AS deps
RUN apk --no-cache update
RUN apk add pkgconfig opus-dev opusfile-dev build-base
WORKDIR /project
COPY go.mod ./
COPY go.sum ./
RUN go mod download

FROM deps AS builder
ARG CMD_NAME
COPY . ./
RUN go build -ldflags="-w" -o run ./cmd/$CMD_NAME

FROM alpine
RUN apk --no-cache update
RUN apk add opus opusfile
COPY --from=builder /project/run /bin/run
ENTRYPOINT [ "run" ]

