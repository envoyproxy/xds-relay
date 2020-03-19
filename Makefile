SERVICE_NAME := xds-relay
GOREPO := ${GOPATH}/src/github.com/envoyproxy/xds-relay

# Compiles the binary and installs it into /usr/local/bin
.PHONY: compile
compile:
	mkdir -p ./bin && \
	  go build -o ${GOREPO}/bin/${SERVICE_NAME} && \
	  cp ${GOREPO}/bin/${SERVICE_NAME} /usr/local/bin/${SERVICE_NAME} && \
	  cd ${GOREPO}/cmd/configuration-validator && \
	  go build -o ${GOREPO}/bin/configuration-validator

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

# Run golangci-lint
.PHONY: lint
lint:
	golangci-lint run
