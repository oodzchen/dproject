#!/usr/bin/env bash

# source ./env.sh 

# echo "Init SQL script"
# echo "Database: $DB_NAME"
# echo "User: $DB_USER"

# sleep 3s

cat $1 | sed "s/PLACEHOLD_ADMIN_PASSWORD/$ADMIN_PASSWORD/g" \
| sed "s/PLACEHOLD_DB_NAME/$DB_NAME/g" \
| sed "s/PLACEHOLD_DB_USER/$DB_USER/g"
