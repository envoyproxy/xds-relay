## Synopsis

configuration-validator is a CLI tool used to validate xds-relay configuration
files.

The tool can be embedded in user CI pipelines to ensure that xds-relay is
provisioned correctly.

It currently supports:

* Aggregation Key configurations

## Aggregation Key

The aggregation key yaml configuration is validated against the
[proto file](https://github.com/envoyproxy/xds-relay/blob/master/api/protos/aggregation/v1/aggregation.proto
).  The same proto file is annotated with [protoc-gen-validate constraint
rules](https://github.com/envoyproxy/protoc-gen-validate/#constraint-rules) to
enforce semantic rules. Both the xds-relay initialization and the CLI share the
same validation method.

### Usage

```
configuration-validator aggregation [flags]
```

#### Options

```
  -p, --path string   path to aggregation key configuration file
  -h, --help          help for configuration-validator
```
