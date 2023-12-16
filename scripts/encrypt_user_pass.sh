#!/usr/bin/env bash

env_file=${1:-./.env.local.dev}

source ./scripts/init_env.sh $env_file

hash_password() {
    htpasswd -bnBC 10 "" $1 | tr -d ":\n" 
}

encrypt_password(){
    hash_password "$USER_DEFAULT_PASSWORD" > ./config/db/initdb/encrypted_user_password.txt
}

encrypt_password
