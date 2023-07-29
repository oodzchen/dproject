#!/usr/bin/env bash

GO_PATH=$(go env GOPATH)
SQL_FILE="$GO_PATH/src/github.com/oodzchen/dproject/config/seed.sql"
CONTAINER_NAME="disc_db"
POSTGRES_USER="postgres"
DEFAULT_DB_NAME="postgres"

echo "----------------------------- seed sample data -----------------------------"
docker cp "$SQL_FILE" "$CONTAINER_NAME":/tmp/data.sql

docker exec -i "$CONTAINER_NAME" psql -U postgres -d "$DEFAULT_DB_NAME" -f /tmp/data.sql

echo "----------------------------- encrypt password -----------------------------"
DB_NAME="discuss"
DB_USER="admin"

function hashpassword {
    htpasswd -bnBC 10 "" "$1" | tr -d ":\n" 
}

CSV_FILE="$GO_PATH/src/github.com/oodzchen/dproject/config/data.csv "
# echo "csv file path: $CSV_FILE"
if test -e $CSV_FILE; then
    # echo "remove existing $CSV_FILE..."
    rm $CSV_FILE
fi

touch $CSV_FILE

docker exec -i "$CONTAINER_NAME" psql -d "$DB_NAME" -U "$DB_USER" -t -c "select id, password from users;" | sed -e '${/./!d;}' |
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

docker cp $CSV_FILE "$CONTAINER_NAME:/tmp/data.csv"

docker exec -i "$CONTAINER_NAME" psql -d "$DB_NAME" -U "$DB_USER" -t \
       -c "create temp table tmp_u (id int, password text);\
copy tmp_u from '/tmp/data.csv' with (format csv, delimiter ',');\
update users set password = tmp_u.password from tmp_u where users.id = tmp_u.id;\
drop table tmp_u;"

rm $CSV_FILE
