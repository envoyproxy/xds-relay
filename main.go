package main

import (
	"io/ioutil"
	"log"

	"github.com/envoyproxy/xds-relay/internal/app/server"
	yamlproto "github.com/envoyproxy/xds-relay/internal/pkg/util"
	aggregationv1 "github.com/envoyproxy/xds-relay/pkg/api/aggregation/v1"
	bootstrapv1 "github.com/envoyproxy/xds-relay/pkg/api/bootstrap/v1"
	"github.com/spf13/cobra"
)

var (
	cfgFile          string
	aggregationRules string
	logLevel         string

	bootstrapCmd = &cobra.Command{
		Use: "xds-relay",
		Run: func(cmd *cobra.Command, args []string) {
			bootstrapConfigFileContent, err := ioutil.ReadFile(cfgFile)
			if err != nil {
				log.Fatal(err)
			}
			var bootstrapConfig bootstrapv1.Bootstrap
			err = yamlproto.FromYAMLToBootstrapConfiguration(string(bootstrapConfigFileContent), &bootstrapConfig)
			if err != nil {
				log.Fatal(err)
			}

			aggregationRulesFileContent, err := ioutil.ReadFile(aggregationRules)
			if err != nil {
				log.Fatal(err)
			}
			var aggregationRules aggregationv1.KeyerConfiguration
			err = yamlproto.FromYAMLToKeyerConfiguration(string(aggregationRulesFileContent), &aggregationRules)
			if err != nil {
				log.Fatal(err)
			}

			// TODO: Hand off values to the server/orchestrator creation process (includes config parsing)
			server.Run()
		},
	}
)

func main() {
	bootstrapCmd.Flags().StringVarP(&cfgFile, "config-file", "c", "", "path to bootstrap configuration file")
	bootstrapCmd.Flags().StringVarP(&aggregationRules, "aggregation-rules", "a", "", "path to aggregation rules file")
	bootstrapCmd.Flags().StringVarP(&logLevel, "log-level", "l", "info", "the logging level")
	if err := bootstrapCmd.MarkFlagRequired("config-file"); err != nil {
		log.Fatal("Could not mark the config-file flag as required: ", err)
	}
	if err := bootstrapCmd.MarkFlagRequired("aggregation-rules"); err != nil {
		log.Fatal("Could not mark the aggregation-rules flag as required: ", err)
	}
	if err := bootstrapCmd.Execute(); err != nil {
		log.Fatal("Issue parsing command line: ", err)
	}
}
