#!/usr/bin/env bash

docker compose --env-file ./.env.local.docker.dev -f ./docker-compose.dev.yml up -d

if [ $? -ne 0 ];then
   echo "Docker compose up failed"
   exit 1
fi

set -o allexport
source .env.local.docker.dev
set +o allexport

echo "DB_NAME: $DB_NAME"
echo "DB_USER: $DB_USER"
echo "ADMIN_PASSWORD: $ADMIN_PASSWORD"
echo "DB_HOST: $DB_HOST"
echo "DB_PORT: $DB_PORT"

export SESSION_SECRET=$(uuidgen)
export CSRF_SECRET=$(uuidgen)

cat ./.env.example | envsubst > ./.env.local

if [ $? -ne 0 ];then
   echo "Init env file failed"
   exit 1
fi

cleanup() {
    echo "Ctrl+C pressed. Cleaning up..."
    # Add your cleanup code here
    docker compose -f ./docker-compose.dev.yml down
    exit 1  # Exit the script
}

trap cleanup SIGINT

fresh
