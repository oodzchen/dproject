FROM postgres:15.4

ARG DB_NAME
ARG DB_USER
ARG ADMIN_PASSWORD

WORKDIR /usr/src/app
COPY . .

RUN apt-get update && apt-get install -y --no-install-recommends uuid-runtime apache2-utils

RUN ./a_init.sh ./b_seed.sql > /docker-entrypoint-initdb.d/b_seed.sql

RUN ./a_init.sh ./c_seed.sh > /docker-entrypoint-initdb.d/c_seed.sh
