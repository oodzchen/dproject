# syntax=docker/dockerfile:1

FROM golang:1.20.8 AS build-stage

WORKDIR /app

ARG GOPROXY

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o ./bin/webapp .

FROM alpine:3.18 AS release-stage
WORKDIR /app

COPY --from=build-stage /app .

CMD ["/app/bin/webapp"]
