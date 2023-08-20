#!/usr/bin/env bash

until pg_isready -h "$DB_HOST" -p "$DB_PORT" -U postgres
do
    echo "Waiting for postgres..."
    sleep 2;
done
