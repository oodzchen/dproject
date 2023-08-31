#!/usr/bin/env bash

source ./run_docker.sh $1

echo "env file: $env_file"

echo "is ci: $2"

cleanup() {
    echo "Ctrl+C pressed. Cleaning up..."

    stop_docker
    exit 1
}

if [ "$2" == "ci" ];then
    app &
else
    trap cleanup SIGINT
    fresh
fi
