#!/usr/bin/env bash

env_file=${1:-./.env.local.dev}
compose_file=${2:-./docker-compose.yml}

source ./scripts/init_env.sh $env_file
source ./scripts/run_docker.sh $env_file $compose_file
