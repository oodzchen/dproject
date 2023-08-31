#!/usr/bin/env bash

config_file=./docker-compose.dev.yml

source ./init_env.sh $1

run_docker(){
    # init_env $1
    encrypt_password
    if [ "$2" == "build" ];then
       docker compose --env-file "$1" -f $config_file Pull
       docker compose --env-file "$1" -f $config_file rm -f
       docker compose --env-file "$1" -f $config_file build --no-cache
    fi
    
    docker compose --env-file "$1" -f $config_file up -d
    
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

stop_docker(){
    docker compose -f $config_file down
}

run_docker $env_file
