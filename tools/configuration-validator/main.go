package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"

	yamlproto "github.com/envoyproxy/xds-relay/internal/pkg/util"
	aggregationv1 "github.com/envoyproxy/xds-relay/pkg/api/aggregation/v1"
)

type KeyerConfiguration = aggregationv1.KeyerConfiguration

func main() {
	var configFile string
	flag.StringVar(&configFile, "config", "", "path to configuration file")
	flag.Parse()

	if configFile == "" {
		flag.Usage()
		os.Exit(1)
	}

	yamlFileContent, err := ioutil.ReadFile(configFile)
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
}
