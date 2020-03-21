export SERVICE_NAME=xds-relay
export GOREPO=${GOPATH}/src/github.com/envoyproxy/${SERVICE_NAME}
SOURCE_FILES?=$$(go list ./... | grep -v integration-tests)

.PHONY: setup
setup:
	mkdir -p ${GOREPO}/bin

.PHONY: compile
compile: setup  ## Compiles the binary and installs it into /usr/local/bin
	go build -o ${GOREPO}/bin/${SERVICE_NAME} && \
	  cp ${GOREPO}/bin/${SERVICE_NAME} /usr/local/bin/${SERVICE_NAME}

.PHONY: install
install: ## Installs dependencies
	go mod vendor

.PHONY: unit
unit: ## Run all unit tests with coverage report
	go test -v -cover $(SOURCE_FILES)

.PHONY: integration-tests
integration-tests:  ## Run integration tests
	go test -v ${GOREPO}/integration-tests/

.PHONY: compile-protos
compile-protos: ## Compile proto files
	./scripts/generate-api-protos.sh

.PHONY: compile-validator-tool
compile-validator-tool: setup  ## Compiles configuration validator tool
	cd ${GOREPO}/tool/configuration-validator && \
	  go build -o ${GOREPO}/bin/configuration-validator

.PHONY: lint
lint: ## Run golangci-lint
	golangci-lint run

# Absolutely awesome: http://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := compile
