package main

import (
	"io/ioutil"
	"log"
	"os"

	yamlproto "github.com/envoyproxy/xds-relay/internal/pkg/util"
	aggregationv1 "github.com/envoyproxy/xds-relay/pkg/api/aggregation/v1"

	"github.com/spf13/cobra"
)

type KeyerConfiguration = aggregationv1.KeyerConfiguration

var (
	cfgFile string

	validatorCmd = &cobra.Command{
		Use:   "configuration-validator",
		Short: "A tool to help validate aggregation key configuration files",
		Long: `configuration-validator is a CLI tool used to validate aggregation key yaml files.
The aggregation key yaml file is validated against the
[https://github.com/envoyproxy/xds-relay/blob/master/api/protos/aggregation/v1/aggregation.proto](proto file).
The same proto file is annotated with
[protoc-gen-validate constraint rules](https://github.com/envoyproxy/protoc-gen-validate/#constraint-rules)
to enforce semantic rules. Both the xds-relay initialization and the CLI share the same validation method.

The idea behind having a separate CLI tool to validate yaml files is to enable tooling around the generation
and validation of config files. For example, imagine a scenario where an automated process generates a
config file and a CI pipeline uses the validator tool to ensure the generated yaml is valid.
`,
		Run: func(cmd *cobra.Command, args []string) {
			yamlFileContent, err := ioutil.ReadFile(cfgFile)
			if err != nil {
				log.Fatal(err)
			}

			// Load config file into a valid KeyerConfiguration object and,
			// in case we encounter an error, print the error string to
			// stderr.
			var config KeyerConfiguration
			err = yamlproto.FromYAMLToKeyerConfiguration(string(yamlFileContent), &config)
			if err != nil {
				log.Fatal(err)
			}
			os.Exit(0)
		},
	}
)

func main() {
	validatorCmd.Flags().StringVarP(&cfgFile, "config-file", "c", "", "path to configuration file")
	if err := validatorCmd.MarkFlagRequired("config-file"); err != nil {
		log.Fatal("Could not mark config-file flag as required")
	}

	if err := validatorCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
