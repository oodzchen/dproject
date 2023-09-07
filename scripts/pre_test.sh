#!/usr/bin/env bash

store_testdata=./store/testdata

echo "PG_PASSWORD=$PG_PASSWORD" > ./.env.testing
echo "DB_NAME=$DB_NAME" >> ./.env.testing
echo "DB_USER=$DB_USER" >> ./.env.testing
echo "ADMIN_PASSWORD=$ADMIN_PASSWORD" >> ./.env.testing
echo "USER_DEFAULT_PASSWORD=$USER_DEFAULT_PASSWORD" >> ./.env.testing
echo "DB_HOST=$DB_HOST" >> ./.env.testing
echo "DB_PORT=$DB_PORT" >> ./.env.testing
echo "DEBUG=0" >> ./.env.testing
echo "DOMAIN_NAME=localhost" >> ./.env.testing
echo "PORT=3000" >> ./.env.testing

echo "env file len $(expr length "$(cat ./.env.testing)")"

if [ ! -d $store_testdata ];then
   mkdir $store_testdata
fi

cp ./.env.testing $store_testdata
