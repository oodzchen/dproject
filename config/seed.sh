#!/bin/bash

SQL_FILE="$HOME/go/src/github.com/oodzchen/dproject/config/seed.sql"
CONTAINER_NAME="disc_db2"

POSTGRES_USER="postgres"
DATABASE_NAME="postgres"

docker cp "$SQL_FILE" "$CONTAINER_NAME":/tmp/data.sql

docker exec -it "$CONTAINER_NAME" psql -U postgres -d "$DATABASE_NAME" -f /tmp/data.sql

