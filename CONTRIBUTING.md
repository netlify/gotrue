# CONTRIBUTING

Contributions are always welcome, no matter how large or small. Before contributing,
please read the [code of conduct](CODE_OF_CONDUCT.md).

## Setup

* Install Go 1.16
* Install [Soda CLI](https://gobuffalo.io/en/docs/db/toolbox)
* Install Docker to run tests

GoTrue uses the Go Modules support built into Go 1.11 to build. The easiest is to clone GoTrue in a directory outside of GOPATH, as in the following example:

```sh
$ git clone https://github.com/supabase/gotrue
$ cd gotrue
$ make deps
```

## Building

```sh
$ make build
```

## Running database migrations for supabase
- Start PostgreSQL inside a docker container running `hack/postgresd.sh`
```sh
$ make migrate_supabase
```
- Make sure you change the password of the user `supabase_auth_admin` 
```sh
gotrue_postgres_development=> alter user supabase_auth_admin with password 'PASSWORD';
```
- Create a `.env` file to store the custom gotrue environment variables. You can refer to an example of the `.env` file [here](hack/test.env)
- Set `DATABASE_URL="postgres://supabase_auth_admin:root@localhost:5432/gotrue_postgres_development"` so that gotrue can connect to the database.
- Migrations for the auth schema used will be stored in `migrations_postgres`

### Rollback migration
- To rollback the latest migration created, run the following command. 
```sh
soda migrate down -p migrations_postgres -e postgres_development -c hack/database.yml -d
```

## Running database migrations for development and testing

- Make sure your database can be accessed with user `root` without a password.
- Alternatively, you can start MySQL inside a docker container running `hack/mysqld.sh`.

### Migrations for development

```sh
$ make migrate_dev
```

### Migrations for testing

```sh
$ make migrate_test
```

## Testing

```sh
$ make test
```

## Pull Requests

We actively welcome your pull requests.

1. Fork the repo and create your branch from `master`.
2. If you've added code that should be tested, add tests.
3. If you've changed APIs, update the documentation.
4. Ensure the test suite passes.
5. Make sure your code lints.

## License

By contributing to Netlify CMS, you agree that your contributions will be licensed
under its [MIT license](LICENSE).
