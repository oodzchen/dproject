#!/usr/bin/env bash

env_file=${1:-./.env.local.dev}
compose_file=${2:-./docker-compose.dev.yml}

echo "docker compose config: $compose_file"

run_docker(){
    docker compose --env-file $env_file -f $compose_file up -d
    
    if [ $? -ne 0 ];then
        echo "Docker compose up failed"
        exit 1
    fi
}

rebuild_docker(){
    docker compose -f $compose_file rm -f
    docker compose -f $compose_file build --no-cache
}

stop_docker(){
    docker compose -f $compose_file down
}
