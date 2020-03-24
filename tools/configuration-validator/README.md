## configuration-validator

A tool to help validate aggregation key configuration files

### Synopsis

configuration-validator is a CLI tool used to validate aggregation key yaml files.
The aggregation key yaml file is validated against the proto file defined in https://github.com/envoyproxy/xds-relay/blob/master/api/protos/aggregation/v1/aggregation.proto.
The proto file is annotated with https://github.com/envoyproxy/protoc-gen-validate/#constraint-rules constraint rules to enforce semantic rules, e.g., 
The CLI tool uses the same validation method as the one found in xds-relay initialization. Having a separate tool to validate config files enables external processes to consume and generate only valid files. For example, imagine a scenario where an automated process generates yaml and a CI pipeline uses the validator tool to ensure the generated yaml is valid.

```
configuration-validator [flags]
```

### Options

```
  -c, --config-file string   path to configuration file
  -h, --help                 help for configuration-validator
```
