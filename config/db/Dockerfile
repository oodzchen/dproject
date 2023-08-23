FROM postgres:15.4

# RUN echo "deb http://mirrors.tuna.tsinghua.edu.cn/debian/ bookworm main contrib non-free non-free-firmware" > /etc/apt/sources.list
# RUN echo "deb http://mirrors.tuna.tsinghua.edu.cn/debian/ bookworm-updates main contrib non-free non-free-firmware" >> /etc/apt/sources.list
# RUN echo "deb http://mirrors.tuna.tsinghua.edu.cn/debian/ bookworm-backports main contrib non-free non-free-firmware" >> /etc/apt/sources.list

RUN apt-get update && apt-get install -y --no-install-recommends uuid-runtime apache2-utils

ARG DB_NAME
ARG DB_USER
ARG ADMIN_PASSWORD

WORKDIR /usr/src/app
COPY . .

RUN ./a_init.sh ./b_seed.sql > /docker-entrypoint-initdb.d/b_seed.sql

RUN ./a_init.sh ./c_seed.sh > /docker-entrypoint-initdb.d/c_seed.sh
