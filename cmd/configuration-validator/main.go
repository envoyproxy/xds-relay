package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"

	"github.com/envoyproxy/xds-relay/internal/pkg/util"
	aggregationv1 "github.com/envoyproxy/xds-relay/pkg/api/aggregation/v1"
)

type KeyerConfiguration = aggregationv1.KeyerConfiguration

func main() {
	var yamlFilename string
	flag.StringVar(&yamlFilename, "yaml", "", "configuration yaml filename")
	flag.Parse()

	if yamlFilename == "" {
		flag.Usage()
		os.Exit(1)
	}

	// Load yaml string from flags
	yamlFileContent, err := ioutil.ReadFile(yamlFilename)
	if err != nil {
		// TODO add a more descriptive error message
		log.Fatal(err)
	}

	// Load config file into a valid KeyerConfiguration object and,
	// in case we encounter an error, print the error string to
	// stderr.
	var config KeyerConfiguration
	err = yamlproto.FromYAMLToKeyerConfiguration(string(yamlFileContent), &config)
	if err != nil {
		// TODO add a more descriptive error message. Mention that the validation stops
		// after the first error.
		log.Fatal(err)
	}
}
