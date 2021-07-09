#!/usr/bin/env bash

DB_ENV=$1

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
DATABASE="$DIR/database.yml"

export GOTRUE_DB_DRIVER="postgres"
export GOTRUE_DB_DATABASE_URL="postgres://postgres:root@localhost:5432/$DB_ENV?sslmode=disable"
export GOTRUE_DB_MIGRATIONS_PATH=$DIR/../migrations_postgres

go run main.go migrate -c $DIR/test.env
