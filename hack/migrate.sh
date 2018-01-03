#!/usr/bin/env bash

DB_ENV=$1

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
DATABASE="$DIR/database.yml"

export GOTRUE_DB_DRIVER="mysql"
export GOTRUE_DB_DATABASE_URL="root@tcp(127.0.0.1:3306)/gotrue_$DB_ENV?parseTime=true"

echo soda -v
soda drop -d -e $DB_ENV -c $DATABASE
soda create -d -e $DB_ENV -c $DATABASE
soda migrate -d -e $DB_ENV -c $DATABASE
