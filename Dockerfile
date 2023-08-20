FROM golang:1.20

RUN apt-get update && apt-get install -y --no-install-recommends uuid-runtime apache2-utils

WORKDIR /usr/src/app

# pre-copy/cache go.mod for pre-downloading dependencies and only redownloading them in subsequent builds if they change
COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN go build -v -o /usr/local/bin/app .

ADD ./config/endpoint.sh /usr/local/bin/run.sh

CMD "run.sh" "/usr/src/app" && sleep infinity

EXPOSE 3000/tcp
