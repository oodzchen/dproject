#!/usr/bin/env bash

DEFAULT_PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && cd ../ && pwd)"
PROJECT_ROOT=${1:-$DEFAULT_PROJECT_ROOT}
SQL_FILE="$PROJECT_ROOT/config/seed.sql"
TEMP_SQL_FILE="$PROJECT_ROOT/config/seed_tmp.sql"
ADMIN_PWD_FILE="$PROJECT_ROOT/config/admin_password.txt"
EXAMPLE_ENV_FILE="$PROJECT_ROOT/.env.example"
LOCAL_ENV_FILE="$PROJECT_ROOT/.env.local"

echo "Project root: $PROJECT_ROOT"

# CONTAINER_NAME="$1"
POSTGRES_USER="postgres"
DEFAULT_DB_NAME="postgres"

DB_NAME="discuss"
DB_USER="admin"

ADMIN_PWD=$(cat "$ADMIN_PWD_FILE")
SESSION_SECRET=$(uuidgen)
CSRF_SECRET=$(uuidgen)

echo "Container name: $DB_CONTAINER_NAME"

echo "----------------------------- seed sample data -----------------------------"
cp "$SQL_FILE" "$TEMP_SQL_FILE"
sed -i "s/ADMIN_PASSWORD/$ADMIN_PWD/" "$TEMP_SQL_FILE"

cp -f "$EXAMPLE_ENV_FILE" "$LOCAL_ENV_FILE"
sed -i "s/PLACEHOLD_SESSION_SECRET/$SESSION_SECRET/" "$LOCAL_ENV_FILE"
sed -i "s/PLACEHOLD_CSRF_SECRET/$CSRF_SECRET/" "$LOCAL_ENV_FILE"
sed -i "s/PLACEHOLD_ADMIN_PASSWORD/$ADMIN_PWD/" "$LOCAL_ENV_FILE"
sed -i "s/PLACEHOLD_DB_HOST/$DB_HOST/" "$LOCAL_ENV_FILE"
sed -i "s/PLACEHOLD_DB_PORT/$DB_PORT/" "$LOCAL_ENV_FILE"

docker cp "$TEMP_SQL_FILE" "$DB_CONTAINER_NAME":/tmp/data.sql

if [ $? -eq 0 ]; then
    rm "$TEMP_SQL_FILE"
    echo "Copy SQL file to docker success"
else
    echo "Copy SQL file to docker failed"
    exit 1
fi

docker exec -i "$DB_CONTAINER_NAME" psql -U postgres -d "$DEFAULT_DB_NAME" -f /tmp/data.sql

if [ $? -eq 0 ]; then
    echo "Seed sample data success"
else
    echo "Seed sample data failed"
    exit 1
fi

echo "----------------------------- encrypt password -----------------------------"

echo "Database: $DB_NAME"
echo "User: $DB_USER"

function hashpassword {
    htpasswd -bnBC 10 "" $1 | tr -d ":\n" 
}

CSV_FILE="$PROJECT_ROOT/config/data.csv "

# echo "csv file path: $CSV_FILE"
if test -e $CSV_FILE; then
    # echo "remove existing $CSV_FILE..."
    rm $CSV_FILE
fi

touch $CSV_FILE

{
    docker exec -i "$DB_CONTAINER_NAME" psql -d "$DB_NAME" -U "$DB_USER" -t -c "select id, password from users;" | sed -e '${/./!d;}' |
	while IFS= read -r row; do
	    # echo "$row"
	    user_id=$(echo "$row" | cut -d "|" -f 1 | tr -d ' ')
	    plain_password=$(echo "$row" | cut -d "|" -f 2 | tr -d ' ')

	    # echo "user id: $user_id"

	    echo "user id: $user_id, plain_password: $plain_password"

	    hashed_password=$(hashpassword "$plain_password")

	    # echo "hashed_password: $hashed_password"

	    echo "$user_id,$hashed_password" >> $CSV_FILE
	done

    docker cp $CSV_FILE "$DB_CONTAINER_NAME:/tmp/data.csv"

    docker exec -i "$DB_CONTAINER_NAME" psql -d "$DB_NAME" -U "$DB_USER" -t \
	   -c "create temp table tmp_u (id int, password text);\
copy tmp_u from '/tmp/data.csv' with (format csv, delimiter ',');\
update users set password = tmp_u.password from tmp_u where users.id = tmp_u.id;\
drop table tmp_u;"
}

if [ $? -eq 0 ]; then
    rm $CSV_FILE
    echo "Encrypt password successed"
else
    echo "Encrypt password failed"
    exit 1
fi

app &