#!/usr/bin/env bash

docker rm -f gotrue_mysql >/dev/null 2>/dev/null || true

docker volume inspect mysql_data 2>/dev/null >/dev/null || docker volume create --name mysql_data >/dev/null

docker run --name gotrue_mysql \
	-p 3306:3306 \
	-e MYSQL_ALLOW_EMPTY_PASSWORD=yes \
	--volume mysql_data:/var/lib/mysql \
	-d mysql:5.7 mysqld --bind-address=0.0.0.0
