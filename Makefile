export SERVICE_NAME=xds-relay

# Compiles the binary and installs it into /usr/local/bin
.PHONY: compile
compile:
	mkdir -p ./bin && \
	  go build -o ./bin/${SERVICE_NAME} && \
	  cp ./bin/${SERVICE_NAME} /usr/local/bin/${SERVICE_NAME}

# Installs dependencies
.PHONY: install
install:
	go mod vendor

# Run all unit tests with coverage report
.PHONY: unit
unit:
	go test -v -cover ./...

# Run golangci-lint
.PHONY: lint
lint:
	golangci-lint run
