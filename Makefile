.PHONY: all build deps image lint migrate test vet
CHECK_FILES?=$$(go list ./... | grep -v /vendor/)

DOCKER_DIR=hack
DOCKER_COMPOSE=COMPOSE_DOCKER_CLI_BUILD=1 DOCKER_BUILDKIT=1 docker compose -f ${DOCKER_DIR}/docker-compose.yml

help: ## Show this help.
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {sub("\\\\n",sprintf("\n%22c"," "), $$2);printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

all: vet test build ## Run the tests and build the binary.

build: ## Build the binary.
	go build -ldflags "-X github.com/netlify/gotrue/cmd.Version=`git rev-parse HEAD`"

deps: ## Install dependencies.
	@go install golang.org/x/lint/golint
	@go mod download

image: ## Build the Docker image.
	docker build .

lint: ## Lint the code.
	golint $(CHECK_FILES)

test: ## Run tests.
	$(DOCKER_COMPOSE) up --no-build --detach tigris
	go test -p 1 -v $(CHECK_FILES)

vet: # Vet the code
	go vet $(CHECK_FILES)
