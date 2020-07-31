# Example

TODO: mention `example-management-server` and the need to assume an envoy binary located at `/usr/local/bin/envoy` (mention https://www.getenvoy.io/).

## Requirements

- envoy present in the PATH
- jq

## Steps

First build the example management server:

    make build-example-management-server
    
This will produce a binary called `example-management-server` under the `bin` directory. This binary runs a simple management server based on `go-control-plane` `SnapshotCache`. It produces a batch of xDS data with a random version every 10 seconds.

Open a window on your terminal and simply run:

    ./bin/example-management-server

Next step is to configure the `xds-relay` server. For that we need to provide 2 files: 
  - an aggregation rules file
  - a bootstrap file
  
You'll find one example of each file in this directory, `aggregation-rules.yaml` and `xds-relay-bootstrap.yaml` respectively.

You're now ready to run `xds-relay` locally. Open another window in your terminal and run:

    ./bin/xds-relay -a internal/example/aggregation-rules.yaml -c internal/example/xds-relay-bootstrap.yaml -m serve

As a final step, it's time to connect an envoy client to `xds-relay`. You're going to find an example bootstrap file called `envoy-bootstrap.yaml` that we're going to use to connect envoy to `xds-relay`. Run:

    envoy -c internal/example/envoy-bootstrap.yaml  --service-node xds-relay

Voilà! You now have a management server producing xDS data that's being relayed by `xds-relay` to an envoy client. You can verify that this is working by checking the dynamic configuration via the envoy admin endpoint:

    curl -v 0:19000/config_dump | jq '.configs | (.[1].dynamic_active_clusters | map({"version": .version_info, "cluster": .cluster.name})) as $clusters | {"clusters": $clusters}'
