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
- Create a `.env` file to store the custom gotrue environment variables. You can refer to an example of the `.env` file [here](hack/test.env)
- Start PostgreSQL inside a docker container running `hack/postgresd.sh`
- Build the gotrue binary `make build`
- Execute the binary `./gotrue`
  - gotrue runs any database migrations from `/migrations` on start 

## Testing
- Currently, we don't use a test db. You can just create a new postgres container, make sure docker is running and do:
```sh
$ ./hack/postgresd.sh
$ make migrate_test
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

By contributing to Gotrue, you agree that your contributions will be licensed
under its [MIT license](LICENSE).
