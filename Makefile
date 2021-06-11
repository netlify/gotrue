.PHONY: all build deps image lint migrate test vet
CHECK_FILES?=$$(go list ./... | grep -v /vendor/)
FLAGS?=-ldflags "-X github.com/supabase/gotrue/cmd.Version=`git rev-parse HEAD`"

help: ## Show this help.
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {sub("\\\\n",sprintf("\n%22c"," "), $$2);printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

all: lint vet build ## Run the tests and build the binary.

build: ## Build the binary.
	go build $(FLAGS)
	GOOS=linux GOARCH=arm64 go build $(FLAGS) -o gotrue-arm64

deps: ## Install dependencies.
	@go get -u github.com/gobuffalo/pop/v5/soda
	@go get -u golang.org/x/lint/golint
	@go mod download

image: ## Build the Docker image.
	docker build .

lint: ## Lint the code.
	golint $(CHECK_FILES)

migrate_dev: ## Run database migrations for development.
	hack/migrate.sh development

migrate_supabase: ## Run database migrations for supabase development.
	hack/migrate_postgres.sh postgres_development

migrate_test: ## Run database migrations for test.
	hack/migrate.sh test

test: ## Run tests.
	go test -p 1 -v $(CHECK_FILES)

vet: # Vet the code
	go vet $(CHECK_FILES)
