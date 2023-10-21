#!/usr/bin/env bash

# store_testdata=./store/testdata
config_dir=./config

echo "PG_PASSWORD=$PG_PASSWORD" > ./.env.testing
echo "DB_NAME=$DB_NAME" >> ./.env.testing
echo "DB_USER=$DB_USER" >> ./.env.testing
echo "ADMIN_PASSWORD=$ADMIN_PASSWORD" >> ./.env.testing
echo "USER_DEFAULT_PASSWORD=$USER_DEFAULT_PASSWORD" >> ./.env.testing
echo "DB_HOST=$DB_HOST" >> ./.env.testing
echo "DB_PORT=$DB_PORT" >> ./.env.testing
echo "DEBUG=$DEBUG" >> ./.env.testing
echo "TEST=1" >> ./.env.testing
echo "DOMAIN_NAME=$DOMAIN_NAME" >> ./.env.testing
echo "APP_LOCAL_HOST=$APP_LOCAL_HOST" >> ./.env.testing
echo "APP_PORT=$APP_PORT" >> ./.env.testing
echo "APP_OUTER_PORT=$APP_OUTER_PORT" >> ./.env.testing
echo "NGINX_PORT=$NGINX_PORT" >> ./.env.testing
echo "NGINX_SSL_PORT=$NGINX_SSL_PORT" >> ./.env.testing

echo "REDIS_HOST=$REDIS_HOST" >> ./.env.testing
echo "REDIS_PORT=$REDIS_PORT" >> ./.env.testing
echo "REDIS_USER=$REDIS_USER" >> ./.env.testing
echo "REDIS_PASSWORD=$REDIS_PASSWORD" >> ./.env.testing
echo "REDISCLI_AUTH=$REDISCLI_AUTH" >> ./.env.testing

echo "env file len $(expr length "$(cat ./.env.testing)")"

# if [ ! -d $store_testdata ];then
#    mkdir $store_testdata
# fi

# cp ./.env.testing $store_testdata
cp ./.env.testing $config_dir
