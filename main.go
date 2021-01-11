package main

import (
	"io/ioutil"
	"log"

	"github.com/envoyproxy/xds-relay/internal/app/server"
	"github.com/envoyproxy/xds-relay/internal/pkg/util/yamlproto"
	aggregationv1 "github.com/envoyproxy/xds-relay/pkg/api/aggregation/v1"
	bootstrapv1 "github.com/envoyproxy/xds-relay/pkg/api/bootstrap/v1"
	"github.com/spf13/cobra"
)

var (
	bootstrapConfigFile        string
	aggregationRulesConfigFile string
	logLevel                   string
	mode                       string

	bootstrapCmd = &cobra.Command{
		Use: "xds-relay",
		Run: func(cmd *cobra.Command, args []string) {
			bootstrapConfigFileContent, err := ioutil.ReadFile(bootstrapConfigFile)
			if err != nil {
				log.Fatal("failed to read bootstrap config file: ", err)
			}
			var bootstrapConfig bootstrapv1.Bootstrap
			err = yamlproto.FromYAMLToBootstrapConfiguration(string(bootstrapConfigFileContent), &bootstrapConfig)
			if err != nil {
				log.Fatal("failed to translate bootstrap config: ", err)
			}

			aggregationRulesFileContent, err := ioutil.ReadFile(aggregationRulesConfigFile)
			if err != nil {
				log.Fatal("failed to read aggregation rules file: ", err)
			}
			var aggregationRulesConfig aggregationv1.KeyerConfiguration
			err = yamlproto.FromYAMLToKeyerConfiguration(string(aggregationRulesFileContent), &aggregationRulesConfig)
			if err != nil {
				log.Fatal("failed to translate aggregation rules: ", err)
			}

			switch mode {
			case "serve", "validate":
				server.Run(&bootstrapConfig, &aggregationRulesConfig, logLevel, mode)
			default:
				log.Fatal("unrecognized mode provided: ", mode)
			}
		},
	}
)

func main() {
	bootstrapCmd.Flags().StringVarP(&bootstrapConfigFile, "config-file", "c", "", "path to bootstrap configuration file")
	bootstrapCmd.Flags().StringVarP(&aggregationRulesConfigFile,
		"aggregation-rules", "a", "", "path to aggregation rules file")
	bootstrapCmd.Flags().StringVarP(&logLevel, "log-level", "l", "", "the logging level")
	bootstrapCmd.Flags().StringVarP(&mode, "mode", "m", "serve",
		"operating mode. Set to 'serve' by default to validate user-entered configs and start the server. "+
			"Set to 'validate' to only validate user-entered configs.")
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
