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
		Description: "negative test: negative cache TTL",
		Parameters: []interface{}{
			"./integration/testdata/bootstrap_configuration_negative_cache_ttl.yaml",
			"./integration/testdata/keyer_configuration_complete_tech_spec.yaml",
			"debug",
			true,
			"invalid Cache.Ttl: value must be greater than or equal to 0s",
		},
	},
}

var _ = Describe("Integration tests for bootstrapping xds-relay", func() {
	DescribeTable("table driven integration tests for bootstrapping xds-relay",
		func(bootstrapConfigFile string, aggregationRulesFile string, logLevel string, wantErr bool, errorMessage string) {
			dir, err := os.Getwd()
			Expect(err).To(BeNil())

			// #nosec G204
			cmd := exec.Command(path.Join(dir, "bin", bootstrapBinaryName),
				"-c", bootstrapConfigFile,
				"-a", aggregationRulesFile,
				"-l", logLevel)
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
