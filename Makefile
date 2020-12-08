export SERVICE_NAME=xds-relay

.PHONY: setup
setup:
	mkdir -p ./bin

.PHONY: compile
compile: setup  ## Compiles the binary
	go build -o ./bin/${SERVICE_NAME}

.PHONY: install
install: ## Installs dependencies
	go mod vendor

.PHONY: unit
unit: ## Run all unit tests with coverage report
	go test -v -cover -race ./...

.PHONY: integration-tests
integration-tests:  ## Run integration tests
	go test -tags integration -v ./integration/

.PHONY: e2e-tests
e2e-tests: ## Run e2e tests
	go test -parallel 1 -tags end2end,docker -v ./integration/

.PHONY: compile-protos
compile-protos: ## Compile proto files
	./scripts/generate-api-protos.sh

.PHONY: compile-validator-tool
compile-validator-tool: setup  ## Compiles configuration validator tool
	go build -o ./bin/configuration-validator $$(go list ./tools/configuration-validator)

.PHONY: build-e2e-tests-docker-image
build-e2e-tests-docker-image: ## Build docker image for use in e2e tests
	docker build . --file Dockerfile-e2e-tests --tag xds-relay

.PHONY: lint
lint: ## Run golangci-lint
	golangci-lint run

.PHONY: build-example-management-server
build-example-management-server: compile  ## Build example management server
	go build -o ./bin/example-management-server example/main.go

# Absolutely awesome: http://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
help:
	@grep -E '^[[:alnum:]-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := compile
