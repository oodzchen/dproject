#!/usr/bin/env bash

set -e

env_file_name=${1:-.env.local.dev}
GIT_REPO_PATH="$HOME/go/src/github.com/oodzchen/dproject/"
DOCKER_COMPOSE_PATH="$GIT_REPO_PATH/docker-compose.yml"
ENV_FILE="$GIT_REPO_PATH/$env_file_name"
APP_SERVICE_NAME="webapp"

if [ -z "$APP_VERSION" ]; then
    APP_VERSION="latest"
fi

echo "app version: $APP_VERSION"

cd $GIT_REPO_PATH

echo "Pulling latest code from Git..."
git pull

sed -i 's/APP_VERSION=.*/APP_VERSION=$APP_VERSION/' $ENV_FILE

echo "Pulling latest Docker images..."
docker compose --env-file $ENV_FILE -f $DOCKER_COMPOSE_PATH pull

echo "Restarting Docker Compose services..."
docker compose --env-file $ENV_FILE -f $DOCKER_COMPOSE_PATH up -d $APP_SERVICE_NAME

echo "Deployment completed successfully."

echo "Clean unused images..."
docker image prune -af

echo "Clean unused images successfully"
