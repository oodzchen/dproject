#!/usr/bin/env bash

set -e

GIT_REPO_PATH="$HOME/go/src/github.com/oodzchen/dproject/"
DOCKER_COMPOSE_PATH="$GIT_REPO_PATH/docker-compose.yml"
ENV_FILE="$GIT_REPO_PATH/.env.local.dev"
APP_SERVICE_NAME="webapp"

cd $GIT_REPO_PATH

echo "Pulling latest code from Git..."
git pull

echo "Pulling latest Docker images..."
docker compose --env-file $ENV_FILE -f $DOCKER_COMPOSE_PATH pull

echo "Restarting Docker Compose services..."
docker compose --env-file $ENV_FILE -f $DOCKER_COMPOSE_PATH up -d $APP_SERVICE_NAME

echo "Deployment completed successfully."