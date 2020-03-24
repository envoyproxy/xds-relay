## configuration-validator

A tool to help validate aggregation key configuration files

### Synopsis

configuration-validator is a CLI tool used to validate aggregation key yaml files.
The aggregation key yaml file is validated against the [https://github.com/envoyproxy/xds-relay/blob/master/api/protos/aggregation/v1/aggregation.proto](proto file).
The same proto file is annotated with [protoc-gen-validate constraint rules](https://github.com/envoyproxy/protoc-gen-validate/#constraint-rules) to enforce semantic rules. Both the xds-relay initialization and the CLI share the same validation method. 

The idea behind having a separate CLI tool to validate yaml files is to enable tooling around the generation and validation of config files. For example, imagine a scenario where an automated process generates a config file and a CI pipeline uses the validator tool to ensure the generated yaml is valid.

```
configuration-validator [flags]
```

### Options

```
  -c, --config-file string   path to configuration file
  -h, --help                 help for configuration-validator
```
