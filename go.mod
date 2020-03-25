module github.com/envoyproxy/xds-relay

go 1.14

require (
	github.com/cactus/go-statsd-client/statsd v0.0.0-20200322202804-24fc78943200
	github.com/envoyproxy/go-control-plane v0.9.4
	github.com/envoyproxy/protoc-gen-validate v0.3.0
	github.com/ghodss/yaml v1.0.0
	github.com/golang/protobuf v1.4.0-rc.4
	github.com/onsi/ginkgo v1.12.0
	github.com/onsi/gomega v1.9.0
	github.com/uber-go/tally v3.3.15+incompatible
	go.uber.org/zap v1.14.0
	google.golang.org/grpc v1.27.1
	google.golang.org/protobuf v1.20.1
)
