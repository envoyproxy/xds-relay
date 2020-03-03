export SERVICE_NAME=xds-relay

# Compiles the binary and installs it into /usr/local/bin
.PHONY: compile
compile:
	mkdir -p ./bin && \
	  go build -o ./bin/${SERVICE_NAME} && \
	  cp ./bin/${SERVICE_NAME} /usr/local/bin/${SERVICE_NAME}

# Installs dependencies
.PHONY: install
install: install-protoc
	go mod vendor

# Run all unit tests with coverage report
.PHONY: unit
unit:
	go test -v -cover ./...

.PHONY: install-protoc
install-protoc:
# Rely on homebrew in the case of MacOSX for now since CI is going to be running only Linux
ifeq ($(shell uname -s), Darwin)
	brew install protobuf
else
	apt-get update && apt-get install unzip
	wget -O /tmp/protoc.zip https://github.com/protocolbuffers/protobuf/releases/download/v3.6.1/protoc-3.6.1-linux-x86_64.zip  \
    && unzip /tmp/protoc.zip -d /usr/local/ \
    && rm /tmp/protoc.zip
endif

.PHONY: compile-protos
compile-protos:
	./scripts/generate-api-protos.sh

# Run golangci-lint
.PHONY: lint
lint:
	golangci-lint run
