# CONTRIBUTING

Contributions are always welcome, no matter how large or small. Before contributing,
please read the [code of conduct](CODE_OF_CONDUCT.md).

## Prerequisites

- [Go 1.24+](https://golang.org/dl/)
- [Docker](https://docs.docker.com/get-docker/) (for running MySQL locally)

## Setup

Clone the repo and install dependencies:

```sh
git clone https://github.com/netlify/gotrue
cd gotrue
make deps
```

This installs the `soda` database migration tool, `golangci-lint`, and downloads Go modules.

## Database

GoTrue uses MySQL 5.7 (matching production). The easiest way to run it locally is with Docker:

```sh
hack/mysqld.sh
```

This starts a MySQL 5.7 container (`gotrue_mysql`) on port 3306 with root access and no password. Data is persisted in a Docker volume (`mysql_data`).

On Apple Silicon (M1/M2/M3), the script uses `--platform linux/amd64` to run the x86 image under Rosetta emulation, since MySQL 5.7 has no native ARM64 image.

If you already have MySQL running locally, it needs to be accessible as user `root` without a password on `127.0.0.1:3306`.

### Running migrations

For development:

```sh
make migrate_dev
```

For testing:

```sh
make migrate_test
```

These commands drop and recreate the database, then apply all migrations from the `migrations/` directory.

## Building

```sh
make build
```

This produces a `gotrue` binary in the project root.

## Testing

Run the full test suite:

```sh
make test
```

Tests run sequentially (`-p 1`) to avoid database conflicts. All tests use the `hack/test.env` configuration, which points to the `gotrue_test` database with table names prefixed by `test_`.

## Linting

```sh
make lint    # Run golangci-lint
make vet     # Run go vet
```

Or run everything (lint, vet, test, build):

```sh
make all
```

## Configuration

GoTrue is configured via environment variables prefixed with `GOTRUE_`, or a `.env` file. See the [README](README.md) for all available configuration options.

For local development, `hack/test.env` provides a working configuration with dummy OAuth credentials and a test JWT secret.

## Pull Requests

1. Fork the repo and create your branch from `master`.
2. If you've added code that should be tested, add tests.
3. If you've changed APIs, update the documentation.
4. Ensure the test suite passes.
5. Make sure your code lints.

## License

By contributing to GoTrue, you agree that your contributions will be licensed
under its [MIT license](LICENSE).
