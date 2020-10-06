# xds-relay
Caching, aggregation, and relaying for xDS compliant clients and origin servers

## Contact

* [Slack](https://envoyproxy.slack.com/): Slack, to get invited go [here](https://envoyslack.cncf.io).
  Please join the `#xds-relay` channel for communication regarding this project.

## Contributing

To get started:

* [Contributing guide](CONTRIBUTING.md)

## Example

In this example we're going to run an instance of a management server that emits xDS data every 10 seconds which will be relayed by an instance of `xds-relay` to 2 instances of envoy.

The goal of this example is to have the envoy instances mapped to the same key in `xds-relay`, namely the cache key `cluster1_cds`.

### Requirements

- envoy present in the PATH (Go to https://www.getenvoy.io/ and follow the guide on how to install an envoy version)
- [jq](https://stedolan.github.io/jq/)
- [curl](https://curl.haxx.se/)

### Steps

#### Management Server
First build the example management server:

    make build-example-management-server
    
This will produce a binary called `example-management-server` under the `bin` directory. This binary runs a simple management server based on `go-control-plane` `SnapshotCache`. It produces a batch of xDS data with a random version every 10 seconds.

Open a window on your terminal and simply run:

    ./bin/example-management-server

#### xds-relay instance
Next step is to configure the `xds-relay` server. For that we need to provide 2 files: 
  - an aggregation rules file
  - a bootstrap file
  
You'll find one example of each file in the `example/config-files` directory, `aggregation-rules.yaml` and `xds-relay-bootstrap.yaml` respectively.

You're now ready to run `xds-relay` locally. Open another window in your terminal and run:

    ./bin/xds-relay -a example/config-files/aggregation-rules.yaml -c example/config-files/xds-relay-bootstrap.yaml -m serve

#### Two envoy instances
As a final step, it's time to connect 2 envoy clients to `xds-relay`. If you do not have envoy installed, you can use [getenvoy](https://www.getenvoy.io/install/envoy/) to install the binary for your OS. You're going to find 2 files named `envoy-bootstrap-1.yaml` and `envoy-bootstrap-2.yaml` that we're going to use to connect the envoy instances to `xds-relay`. Open 2 terminal windows and run:

    envoy -c example/config-files/envoy-bootstrap-1.yaml # on the first window
    envoy -c example/config-files/envoy-bootstrap-2.yaml # on the second window

And voilà! You should be seeing logs flowing in both the terminal window where you're running `xds-relay` and on each of the envoy ones. 

### How to validate that `xds-relay` is working?

We expose the contents of the cache in `xds-relay` via an endpoint, so we can use that to verify what are the contents of the cache for the keys being requested by the two envoy clients:

    curl -s 0:6070/cache/cluster1_cds | jq '(.Cache[0].Resp.Resources.Clusters | map({"name": .name})) as $resp_clusters | (.Cache[0].Requests | map({"version_info": .version_info, "node.id": .node.id, "node.cluster": .node.cluster})) as $reqs | {"response": {"version": .Cache[0].Resp.VersionInfo, "clusters": $resp_clusters}, "requests": $reqs}'

Sample result:

``` shellsession
❯ curl -s 0:6070/cache/cluster1_cds | jq '(.Cache[0].Resp.Resources.Clusters | map({"name": .name})) as $resp_clusters | (.Cache[0].Requests | map({"version_info": .version_info, "node.id": .node.id, "node.cluster": .node.cluster})) as $reqs | {"response": {"version": .Cache[0].Resp.VersionInfo, "clusters": $resp_clusters}, "requests": $reqs}'
{
  "response": {
    "version": "v66936",
    "clusters": [
      {
        "name": "cluster-v66936-0"
      },
      {
        "name": "cluster-v66936-1"
      },
      {
        "name": "cluster-v66936-2"
      }
    ],
    "requests": [
      {
        "version_info": "v66936",
        "node.id": "envoy-client-1",
        "node.cluster": "staging"
      },
      {
        "version_info": "v66936",
        "node.id": "envoy-client-2",
        "node.cluster": "staging"
      }
    ]
  }
}
```

Envoy also exposes an endpoint that lets us investigate what's the current state of the configuration data. If we focus solely on the dynamic cluster information being relayed by `xds-relay`, we can use curl to inspect the envoys by running: 

    curl -s 0:19000/config_dump | jq '.configs | (.[1].dynamic_active_clusters | map({"version": .version_info, "cluster": .cluster.name})) as $clusters | {"clusters": $clusters}'

Sample result:

``` shellsession
❯ curl -s 0:19000/config_dump | jq '.configs | (.[1].dynamic_active_clusters | map({"version": .version_info, "cluster": .cluster.name})) as $clusters | {"clusters": $clusters}'
{
  "clusters": [
    {
      "version": "v56870",
      "cluster": "cluster-v56870-0"
    },
    {
      "version": "v56870",
      "cluster": "cluster-v56870-1"
    },
    {
      "version": "v56870",
      "cluster": "cluster-v56870-2"
    }
  ]
}
```

This is a gif of a tmux session demonstrating this example:

![demo](example/xds-relay-demo.gif)
