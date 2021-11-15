.PHONY: all build deps image lint migrate test vet
CHECK_FILES?=$$(go list ./... | grep -v /vendor/)

help: ## Show this help.
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {sub("\\\\n",sprintf("\n%22c"," "), $$2);printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

all: lint vet test build ## Run the tests and build the binary.

build: ## Build the binary.
	go build -ldflags "-X github.com/netlify/gotrue/cmd.Version=`git rev-parse HEAD`"

deps: ## Install dependencies.
	@go install github.com/gobuffalo/pop/v5/soda
	@go install golang.org/x/lint/golint
	@go mod download

image: ## Build the Docker image.
	docker build .

lint: ## Lint the code.
	golint $(CHECK_FILES)

migrate_dev: ## Run database migrations for development.
	hack/migrate.sh development

migrate_test: ## Run database migrations for test.
	hack/migrate.sh test

test: ## Run tests.
	go test -p 1 -v $(CHECK_FILES)

vet: # Vet the code
	go vet $(CHECK_FILES)
