.PONY: all build deps image test

help: ## Show this help.
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {sub("\\\\n",sprintf("\n%22c"," "), $$2);printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

all: test build ## Run the tests and build the binary.

build: ## Build the binary.
	go build

deps: ## Install dependencies.
	go get -u github.com/Masterminds/glide && glide install

image: ## Build the Docker image.
	docker build .

test: ## Run tests.
	go test -v `go list ./... | grep -v /vendor/`
