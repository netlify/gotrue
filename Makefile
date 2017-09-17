.PONY: all build deps image lint test
CHECK_FILES?=$$(go list ./... | grep -v /vendor/)

help: ## Show this help.
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {sub("\\\\n",sprintf("\n%22c"," "), $$2);printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

all: lint vet test build ## Run the tests and build the binary.

build: ## Build the binary.
	go build -ldflags "-X github.com/netlify/gotrue/cmd.Version=`git rev-parse HEAD`"

deps: ## Install dependencies.
	@go get -u github.com/golang/lint/golint
	@go get -u github.com/Masterminds/glide && glide install

image: ## Build the Docker image.
	echo Building gotrue/gotrue:build
	docker build -t gotrue/gotrue:build . -f Dockerfile.build
	docker create --name gotrue-extract gotrue/gotrue:build
	docker cp gotrue-extract:/go/src/github.com/netlify/gotrue/gotrue ./gotrue
	docker rm -f gotrue-extract
	echo Building gotrue/gotrue:latest
	docker build --no-cache -t gotrue/gotrue:latest .
	rm ./gotrue
lint: ## Lint the code
	golint $(CHECK_FILES)

vet: # Vet the code
	go vet $(CHECK_FILES)

test: ## Run tests.
	go test -v $(CHECK_FILES)
