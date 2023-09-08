#!/usr/bin/env bash

env_file=${1:-./.env.local.dev}
compose_file=${2:-./docker-compose.dev.yml}

echo "docker compose config: $compose_file"

run_docker(){
    encrypt_password
    
    # if [ "$2" == "build" ];then
    #    docker compose --env-file $env_file -f $compose_file pull
    #    docker compose --env-file $env_file -f $compose_file rm -f
    #    docker compose --env-file $env_file -f $compose_file build --no-cache
    # fi
    
    docker compose --env-file $env_file -f $compose_file up -d
    
    if [ $? -ne 0 ];then
        echo "Docker compose up failed"
        exit 1
    fi
}

hash_password() {
    htpasswd -bnBC 10 "" $1 | tr -d ":\n" 
}

encrypt_password(){
    hash_password "$USER_DEFAULT_PASSWORD" > ./config/db/initdb/encrypted_user_password.txt
}

rebuild_docker(){
    docker compose -f $compose_file rm -f
    docker compose -f $compose_file build
}

stop_docker(){
    docker compose -f $compose_file down
}
