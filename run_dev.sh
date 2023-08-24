#!/usr/bin/env bash

env_file="${1:-./.env.local.dev}"

echo "env file: $env_file"

echo "is ci: $2"

run_docker(){
    docker compose --env-file "$1" -f ./docker-compose.dev.yml pull
    docker compose --env-file "$1" -f ./docker-compose.dev.yml rm -f
    
    if [ "$2" == "ci" ];then
    docker compose --env-file "$1" -f ./docker-compose.dev.yml build --no-cache
    fi
    
    docker compose --env-file "$1" -f ./docker-compose.dev.yml up -d
    
    if [ $? -ne 0 ];then
        echo "Docker compose up failed"
        exit 1
    fi
}

function hash_password {
    htpasswd -bnBC 10 "" $1 | tr -d ":\n" 
}

encrypt_password(){
    hash_password "$USER_DEFAULT_PASSWORD" > ./config/db/initdb/encrypted_user_password.txt
}

init_env(){
    set -o allexport
    source $1
    set +o allexport
}

init_app_env() {
    export SESSION_SECRET=$(uuidgen)
    
    export CSRF_SECRET=$(uuidgen)
    
    cat ./.env.example | envsubst > ./.env.local
    if [ $? -ne 0 ];then
        echo "Init env file failed"
        exit 1
    fi
}

cleanup() {
    echo "Ctrl+C pressed. Cleaning up..."

    docker compose -f ./docker-compose.dev.yml down
    exit 1
}

init_env "$env_file"
encrypt_password
run_docker "$env_file"
init_app_env

if [ "$2" == "ci" ];then
    app &
else
    trap cleanup SIGINT
    fresh
fi
