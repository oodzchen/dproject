#!/usr/bin/env bash

echo "----------------------------- encrypt password -----------------------------"

DB_NAME=PLACEHOLD_DB_NAME
DB_USER=PLACEHOLD_DB_USER

echo "Database: $DB_NAME"
echo "User: $DB_USER"

function hashpassword {
    htpasswd -bnBC 10 "" $1 | tr -d ":\n" 
}

CSV_FILE="/tmp/data.csv "

# echo "csv file path: $CSV_FILE"
if test -e $CSV_FILE; then
    # echo "remove existing $CSV_FILE..."
    rm $CSV_FILE
fi

# echo "Who am i: $(whoami)"

touch $CSV_FILE

{
    psql -d "$DB_NAME" -U "$DB_USER" -t -c "select id, password from users;" | sed -e '${/./!d;}' |
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

    psql -d "$DB_NAME" -U "$DB_USER" -t \
	 -c "create temp table tmp_u (id int, password text);\
copy tmp_u from stdin with (format csv, delimiter ',');\
update users set password = tmp_u.password from tmp_u where users.id = tmp_u.id;\
drop table tmp_u;" < $CSV_FILE
}

if [ $? -eq 0 ]; then
    rm $CSV_FILE
    echo "Encrypt password successed"
else
    echo "Encrypt password failed"
    exit 1
fi
