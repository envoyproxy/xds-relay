// +build integration

package integration

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

const (
	binaryName            = "configuration-validator"
	aggregationSubcommand = "aggregation"
)

func TestMain(m *testing.M) {
	// We rely on this simple method of finding the root of the repo to be able to run the tests, as
	// opposed to using a more elaborate method like setting an environment variable.
	err := os.Chdir("..")
	if err != nil {
		fmt.Printf("could not change dir: %v", err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func TestConfigurationValidatorTool(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "configuration-validator integration tests suite")
}

var testCases = []TableEntry{
	{
		Description: "a small positive test",
		Parameters: []interface{}{
			"./integration/testdata/keyer_configuration_request_type_match_string_fragment.yaml",
			false,
			"",
		},
	},
	{
		Description: "a more comprehensive positive test - copied from the tech spec",
		Parameters: []interface{}{
			"./integration/testdata/keyer_configuration_complete_tech_spec.yaml",
			false,
			"",
		},
	},
	{
		Description: "a negative test",
		Parameters: []interface{}{
			"./integration/testdata/keyer_configuration_missing_match_predicate.yaml",
			true,
			"invalid KeyerConfiguration.Fragments[0]: embedded message failed validation | caused by: " +
				"invalid KeyerConfiguration_Fragment.Rules[0]: embedded message failed " +
				"validation | caused by: invalid KeyerConfiguration_Fragment_Rule.Match: value is " +
				"required",
		},
	},
	{
		Description: "a negative test containing an inexistent file",
		Parameters: []interface{}{
			"bogus-file.yaml",
			true,
			"open bogus-file.yaml: no such file or directory",
		},
	},
}

var _ = Describe("Integration tests for the validator tool", func() {
	DescribeTable("table driven integration tests for the validator tool",
		func(ymlFilename string, wantErr bool, errorMessage string) {
			dir, err := os.Getwd()
			Expect(err).To(BeNil())

			// #nosec G204
			cmd := exec.Command(path.Join(dir, "bin", binaryName), aggregationSubcommand, "--path", ymlFilename)
			output, err := cmd.CombinedOutput()
			if wantErr {
				Expect(err).NotTo(BeNil())
				// There is an extra newline in the output.
				Expect(string(output)).Should(HaveSuffix(errorMessage + "\n"))
			} else {
				Expect(err).To(BeNil())
			}
		}, testCases...)

	It("should fail if --config-file flag is missing", func() {
		dir, err := os.Getwd()
		Expect(err).To(BeNil())

		// #nosec G204
		cmd := exec.Command(path.Join(dir, "bin", binaryName), aggregationSubcommand)
		output, _ := cmd.CombinedOutput()
		Expect(string(output)).Should(HavePrefix("Error: required flag(s) \"path\" not set"))
	})
})
