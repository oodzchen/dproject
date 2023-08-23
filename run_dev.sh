#!/usr/bin/env bash

env_file=${1:-./.env.local.docker.dev}   

echo "env file: $env_file"
# echo "env file content: $(cat $env_file)"

echo "is ci: $2"

docker compose -f ./docker-compose.dev.yml pull
docker compose -f ./docker-compose.dev.yml rm -f

if [ "$2" == "ci" ];then
docker compose --env-file "$env_file" -f ./docker-compose.dev.yml build --no-cache
fi

docker compose --env-file "$env_file" -f ./docker-compose.dev.yml up -d

if [ $? -ne 0 ];then
    echo "Docker compose up failed"
    exit 1
fi

set -o allexport
source $env_file
set +o allexport

export SESSION_SECRET=$(uuidgen)

export CSRF_SECRET=$(uuidgen)

cat ./.env.example | envsubst > ./.env.local

if [ $? -ne 0 ];then
    echo "Init env file failed"
    exit 1
fi

cleanup() {
    echo "Ctrl+C pressed. Cleaning up..."

    docker compose -f ./docker-compose.dev.yml down
    exit 1
}

if [ "$2" == "ci" ];then
    app &
else
    trap cleanup SIGINT
    fresh
fi
