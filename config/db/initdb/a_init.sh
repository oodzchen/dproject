#!/usr/bin/env bash

parent_path=$( cd "$(dirname "${BASH_SOURCE[0]}")" ; pwd -P )

encrypted_user_password=$( cat $parent_path/encrypted_user_password.txt )

# echo "Init SQL script"
# echo "Parent path is $parent_path"
# echo "Database: $DB_NAME"
# echo "User: $DB_USER"

psql -v db_name="$DB_NAME" \
     -v db_user="$DB_USER" \
     -v admin_password="'$ADMIN_PASSWORD'" \
     -v user_default_password="'$encrypted_user_password'" \
     -f $parent_path/b_seed.sql.tpl