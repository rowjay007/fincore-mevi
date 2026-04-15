# syntax=docker/dockerfile:1.7

FROM golang:1.26-alpine AS build

RUN apk add --no-cache ca-certificates

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG SERVICE_PKG
ARG BIN_NAME

ENV CGO_ENABLED=0

RUN test -n "$SERVICE_PKG" && test -n "$BIN_NAME"
RUN go build -trimpath -ldflags "-s -w" -o "/out/${BIN_NAME}" "${SERVICE_PKG}"


FROM alpine:3.20

RUN apk add --no-cache ca-certificates

ARG BIN_NAME

COPY --from=build "/out/${BIN_NAME}" "/usr/local/bin/app"

USER 65532:65532

ENTRYPOINT ["/usr/local/bin/app"]
