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
		Short: "A tool to help validate xds-relay configuration files",
	}

	aggregationCmd = &cobra.Command{
		Use:   "aggregation",
		Short: "A tool to help validate xds-relay aggregation key configuration files",
		Long: `aggregation is a comand used to validate aggregation key config files.

The aggregation key yaml file is validated against the
[https://github.com/envoyproxy/xds-relay/blob/master/api/protos/aggregation/v1/aggregation.proto](proto file).
The same proto file is annotated with
[protoc-gen-validate constraint rules](https://github.com/envoyproxy/protoc-gen-validate/#constraint-rules)
to enforce semantic rules. Both the xds-relay initialization and the CLI share the same validation method.
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
		},
	}
)

func main() {
	aggregationCmd.Flags().StringVarP(&cfgFile, "path", "p", "", "path to aggregation key configuration file")
	if err := aggregationCmd.MarkFlagRequired("path"); err != nil {
		log.Fatal("Could not mark the path flag as required")
	}

	validatorCmd.AddCommand(aggregationCmd)
	if err := validatorCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
