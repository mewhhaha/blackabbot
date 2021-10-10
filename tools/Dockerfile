FROM golang:1.17.1 AS deps
RUN apt-get update
RUN apt-get install -y pkg-config libopus-dev libopusfile-dev zip patchelf
WORKDIR /project
COPY go.mod ./
COPY go.sum ./
RUN go mod download

FROM deps AS builder
ENV GOOS linux
ENV GOARCH amd64
CMD ["make", "build"]

