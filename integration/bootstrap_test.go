// +build integration

package integration

import (
	"os"
	"os/exec"
	"path"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

const (
	bootstrapBinaryName = "xds-relay"
)

func TestBootstrap(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "bootstrap integration tests suite")
}

var bootstrapTestCases = []TableEntry{
	{
		Description: "positive test: validate mode",
		Parameters: []interface{}{
			"./integration/testdata/bootstrap_configuration_complete_tech_spec.yaml",
			"./integration/testdata/keyer_configuration_complete_tech_spec.yaml",
			"debug",
			"validate",
			false,
			"",
		}},
	{
		Description: "negative test: invalid mode",
		Parameters: []interface{}{
			"./integration/testdata/bootstrap_configuration_complete_tech_spec.yaml",
			"./integration/testdata/keyer_configuration_complete_tech_spec.yaml",
			"debug",
			"foo",
			true,
			"unrecognized mode provided: foo",
		},
	},
	{
		Description: "negative test: invalid config filepath",
		Parameters: []interface{}{
			"./integration/testdata/foo.yaml",
			"./integration/testdata/keyer_configuration_complete_tech_spec.yaml",
			"debug",
			"foo",
			true,
			"open ./integration/testdata/foo.yaml: no such file or directory",
		},
	},
	{
		Description: "negative test: negative cache TTL",
		Parameters: []interface{}{
			"./integration/testdata/bootstrap_configuration_negative_cache_ttl.yaml",
			"./integration/testdata/keyer_configuration_complete_tech_spec.yaml",
			"debug",
			"serve",
			true,
			"invalid Cache.Ttl: value must be greater than or equal to 0s",
		},
	},
	{
		Description: "negative test: missing address field in address proto",
		Parameters: []interface{}{
			"./integration/testdata/bootstrap_configuration_missing_address.yaml",
			"./integration/testdata/keyer_configuration_complete_tech_spec.yaml",
			"debug",
			"serve",
			true,
			"invalid SocketAddress.Address: value must be a valid hostname, or ip address",
		},
	},
	{
		Description: "negative test: invalid hostname value",
		Parameters: []interface{}{
			"./integration/testdata/bootstrap_configuration_invalid_hostname_value.yaml",
			"./integration/testdata/keyer_configuration_complete_tech_spec.yaml",
			"debug",
			"serve",
			true,
			"invalid SocketAddress.Address: value must be a valid hostname, or ip address",
		},
	},
	{
		Description: "negative test: invalid port value",
		Parameters: []interface{}{
			"./integration/testdata/bootstrap_configuration_invalid_port_value.yaml",
			"./integration/testdata/keyer_configuration_complete_tech_spec.yaml",
			"debug",
			"serve",
			true,
			"invalid SocketAddress.PortValue: value must be less than or equal to 65535",
		},
	},
	{
		Description: "negative test: invalid log level",
		Parameters: []interface{}{
			"./integration/testdata/bootstrap_configuration_invalid_log_level.yaml",
			"./integration/testdata/keyer_configuration_complete_tech_spec.yaml",
			"debug",
			"serve",
			true,
			"invalid value for enum type: \"foo\"",
		},
	},
	{
		Description: "negative test: missing root_prefix",
		Parameters: []interface{}{
			"./integration/testdata/bootstrap_configuration_missing_root_prefix.yaml",
			"./integration/testdata/keyer_configuration_complete_tech_spec.yaml",
			"debug",
			"serve",
			true,
			"caused by: invalid Statsd.RootPrefix: value length must be at least 1 bytes",
		},
	},
	{
		Description: "negative test: missing flush_interval",
		Parameters: []interface{}{
			"./integration/testdata/bootstrap_configuration_missing_flush_interval.yaml",
			"./integration/testdata/keyer_configuration_complete_tech_spec.yaml",
			"debug",
			"serve",
			true,
			"caused by: invalid Statsd.FlushInterval: value is required",
		},
	},
	{
		Description: "negative test: invalid statsd sink port",
		Parameters: []interface{}{
			"./integration/testdata/bootstrap_configuration_invalid_statsd_port.yaml",
			"./integration/testdata/keyer_configuration_complete_tech_spec.yaml",
			"debug",
			"serve",
			true,
			"caused by: invalid SocketAddress.PortValue: value must be less than or equal to 65535",
		},
	},
	{
		Description: "negative test: missing statsd sink address",
		Parameters: []interface{}{
			"./integration/testdata/bootstrap_configuration_missing_statsd_address.yaml",
			"./integration/testdata/keyer_configuration_complete_tech_spec.yaml",
			"debug",
			"serve",
			true,
			"invalid SocketAddress.Address: value must be a valid hostname, or ip address",
		},
	},
}

var _ = Describe("Integration tests for bootstrapping xds-relay", func() {
	DescribeTable("table driven integration tests for bootstrapping xds-relay",
		func(bootstrapConfigFile string, aggregationRulesFile string, logLevel string, mode string, wantErr bool, errorMessage string) {
			dir, err := os.Getwd()
			Expect(err).To(BeNil())

			// #nosec G204
			cmd := exec.Command(path.Join(dir, "bin", bootstrapBinaryName),
				"-c", bootstrapConfigFile,
				"-a", aggregationRulesFile,
				"-l", logLevel,
				"-m", mode)
			output, err := cmd.CombinedOutput()
			if wantErr {
				Expect(err).NotTo(BeNil())
				// There is an extra newline in the output.
				Expect(string(output)).Should(HaveSuffix(errorMessage + "\n"))
			} else {
				Expect(err).To(BeNil())
			}
		}, bootstrapTestCases...)
})
