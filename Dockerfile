# syntax=docker/dockerfile:1

FROM golang:1.20.8 AS build-stage

WORKDIR /app

ARG GOPROXY

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN mkdir -p ./dist/bin
RUN CGO_ENABLED=0 GOOS=linux go build -o ./dist/bin ./...
RUN cp -r ./views ./dist

FROM alpine:3.18 AS release-stage
WORKDIR /app

RUN apk update && apk add curl

COPY --from=build-stage /app/dist .

CMD ["/app/bin/dproject"]
