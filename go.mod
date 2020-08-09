module github.com/envoyproxy/xds-relay

go 1.14

replace github.com/spf13/viper => github.com/spf13/viper v1.7.1

require (
	github.com/cactus/go-statsd-client/statsd v0.0.0-20200322202804-24fc78943200
	github.com/envoyproxy/go-control-plane v0.9.6-0.20200609173151-1f0d489b127f
	github.com/envoyproxy/protoc-gen-validate v0.3.0
	github.com/ghodss/yaml v1.0.0
	github.com/golang/groupcache v0.0.0-20200121045136-8c9f03a8e57e
	github.com/golang/protobuf v1.4.0-rc.4
	github.com/onsi/ginkgo v1.12.0
	github.com/onsi/gomega v1.9.0
	github.com/spf13/cobra v1.0.0
	github.com/stretchr/testify v1.5.1
	github.com/uber-go/tally v3.3.15+incompatible
	go.uber.org/zap v1.15.0
	golang.org/x/tools v0.0.0-20200527150044-688b3c5d9fa5 // indirect
	google.golang.org/grpc v1.27.1
	google.golang.org/protobuf v1.20.1
)
