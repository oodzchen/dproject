#!/bin/sh

redis_dir=/usr/local/etc/redis

if [ ! -d $redis_dir ];then
    mkdir -p $redis_dir
    echo "create redis dir success"
fi

sed 's/\$REDIS_USER/'"$REDIS_USER"'/g; s/\$REDIS_PASSWORD/'"$REDIS_PASSWORD"'/g; s/\$REDISCLI_AUTH/'"$REDISCLI_AUTH"'/g' /app/users.acl.tpl > $redis_dir/users.acl

redis-server /usr/local/etc/redis/redis.conf
