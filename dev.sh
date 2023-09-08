#!/usr/bin/env bash

env_file=${1:-./.env.local.dev}
compose_file=${2:-./docker-compose.dev.yml}

# echo "compose file: $compose_file"

export DEV=1

source ./scripts/init_env.sh $env_file
source ./scripts/run_docker.sh $env_file $compose_file
source ./scripts/pre_test.sh 

cleanup() {
    echo "Ctrl+C pressed. Cleaning up..."

    stop_docker
    exit 1
}

run_docker
trap cleanup SIGINT
fresh
