SERVICE_NAME := xds-relay
GOREPO := ${GOPATH}/src/github.com/envoyproxy/xds-relay

.PHONY: setup
setup:
	mkdir -p ${GOREPO}/bin

# Compiles the binary and installs it into /usr/local/bin
.PHONY: compile
compile: setup
	go build -o ${GOREPO}/bin/${SERVICE_NAME} && \
	  cp ${GOREPO}/bin/${SERVICE_NAME} /usr/local/bin/${SERVICE_NAME}

# Installs dependencies
.PHONY: install
install:
	go mod vendor

# Run all unit tests with coverage report
.PHONY: unit
unit:
	go test -v -cover ./...

# Compile proto files
.PHONY: compile-protos
compile-protos:
	./scripts/generate-api-protos.sh

.PHONY: compile-validator-tool
compile-validator-tool: setup  # Compiles validator tool
	cd ${GOREPO}/cmd/configuration-validator && \
	  go build -o ${GOREPO}/bin/configuration-validator

# Run golangci-lint
.PHONY: lint
lint:
	golangci-lint run
