#!/usr/bin/env bash

env_file="${1:-./.env.local.dev}"
store_testdata=./store/testdata

echo "env file: $env_file"

init_env(){
    set -o allexport
    source $1
    set +o allexport
}

init_app_env() {
    export SESSION_SECRET=$(uuidgen)
    
    export CSRF_SECRET=$(uuidgen)
    
    cat ./.env.example | envsubst > ./.env.local

    if [ ! -d $store_testdata ];then
       mkdir $store_testdata
    fi
       
    cp ./.env.local $store_testdata
       
    if [ $? -ne 0 ];then
        echo "Init env file failed"
        exit 1
    fi
}

init_env $env_file
init_app_env
