#!/usr/bin/env bash

env_file=./.env.local.dev
compose_file=./docker-compose.yml

source ./scripts/init_env.sh $env_file
source ./scripts/run_docker.sh $env_file $compose_file

if [ "$1" = "stop" ];then
   stop_docker
   elif [ "$1" = "rebuild" ];then
	rebuild_docker
	run_docker
   else
       run_docker
fi
