# xds-relay
Caching, aggregation, and relaying for xDS compliant clients and origin servers

## Contact

* [Slack](https://envoyproxy.slack.com/): Slack, to get invited go [here](https://envoyslack.cncf.io).
  Please join the `#xds-relay` channel for communication regarding this project.

## Contributing

To get started:

* [Contributing guide](CONTRIBUTING.md)

## Design

## Aggregation Rules

xds-relay uses aggregation rules to determine how to aggregate inbound connections to the origin
management server ([example](https://github.com/envoyproxy/xds-relay/blob/c83a956952c3e9762a08420ffc1c6bda5e0fff04/integration/testdata/keyer_configuration_e2e.yaml)
).

xDS requests that match under the same set of aggregation rules will fall into the same cache
bucket. The cache is a mapping of a aggregation key to a xDS [DiscoveryResponse](https://www.envoyproxy.io/docs/envoy/latest/api-v2/api/v2/discovery.proto#discoveryresponse)
. This implies that all inbound xDS requests that map to the same aggregation key will retrieve the
same DiscoveryResponse. This is useful in multi-cluster scenarios where xDS configuration does not
differ greatly between clusters. For example, a service application scaled across 100 hosts will
likely require the same data plane configurations across all hosts.

At a high level, an aggregation rule is formed from `fragments`, `rules`, `match`, and `result`s.

#### Fragments
A final aggregation key may look like `fooservice_production_lds`. Each of those identifiers
separated by `_` is what we call a fragment. _fooservice_, _production_, and _lds_ are 3
fragments in the previous example.

#### Rules
Each `fragment` can have multiple `rule`s ([example](https://github.com/envoyproxy/xds-relay/blob/c83a956952c3e9762a08420ffc1c6bda5e0fff04/integration/testdata/keyer_configuration_e2e.yaml#L28-L52)
). A rule defines how to generate a fragment from an xDS [DiscoveryRequest](https://www.envoyproxy.io/docs/envoy/latest/api-v2/api/v2/discovery.proto#discoveryrequest)
. The first matching rule applies, and the rest are ignored.

A rule is formed from a `match` and a `result`. `match` takes a DiscoveryRequest and matches on
some condition. `result` indicates how to generate the fragment. The snippet below illustrates
possible fragment rules:
```
fragments:
  - rules:
      - match:
          request_type_match:
            types:
              - "type.googleapis.com/envoy.api.v2.Listener"
              - "type.googleapis.com/envoy.api.v2.Cluster"
        result:
          request_node_fragment:
            field: 1
            action:
              regex_action: { pattern: "^*-(.*)-*$", replace: "$1" }
```
This example specifies that if the DiscoveryRequest type matches LDS or CDS, to generate the first
fragment from the DiscoveryRequest node ID field by performing a regex replace operation using the
pattern: `"^(.*)-*$"`, and replacing it with the first regex group match `"$1"`. More concretely,
if the DiscoveryRequest had a node ID `1a-fooservice-production`, this would create a fragment with
the string value `fooservice`.

#### Match Predicates
xds-relay current support a limited set of match predicates for rules. More will be added in the
future on a as needed basis. Refer to the [API protos](https://github.com/envoyproxy/xds-relay/blob/master/api/protos/aggregation/v1/aggregation.proto)
for the source of truth.

##### `request_type_match`
- Match on the string value of the xDS request type URL.
- Ex:
  ```
  request_type_match:
    types:
      - "type.googleapis.com/envoy.api.v2.Listener"
  ```

##### `request_node_match`
- Match on fields from the [Node](https://www.envoyproxy.io/docs/envoy/latest/api-v2/api/v2/core/base.proto#envoy-api-msg-core-node)
  object of a DiscoveryRequest.
- Currently supported Node field types and identifiers:
  ```
  id = 0
  cluster = 1
  locality.region = 2
  locality.zone = 3;
  locality.subzone = 4;
  ```
- Ex:
  ```
  request_node_match:
    field: 0
    exact_match: "canary"
  ```
- _Note:_ match can be of type _exact_match_ or _regex_match_. The latter expects a regex pattern.

##### `and_match`
- Conditionally matches on 2 or more predicates. Match condition only applies if all predicates match.
- Ex:
  ```
  and_match:
    rules:
      - request_type_match:
          types:
            - "type.googleapis.com/envoy.api.v2.Listener"
      - request_node_match:
          field: 0
          exact_match: "canary"
  ```

#### Result Predicates
xds-relay current support a limited set of result predicates for rules. More will be added in the
future on a as needed basis. Refer to the [API protos](https://github.com/envoyproxy/xds-relay/blob/master/api/protos/aggregation/v1/aggregation.proto)
for the source of truth.

##### Actions
- Actions can be either a `regex_action` or `exact`. Exact uses the result predicate as is, and
  regex is a match and replace operation on the predicate.
- Ex:
  ```
  action:
    regex_action: { pattern: "^(.*)-*$", replace: "$1" }
  ```
- Ex 2:
  ```
  action: { exact: true }
  ```

##### `request_node_fragment`
- Rules for generating the resulting fragment from a xDS DiscoveryRequest [Node](https://www.envoyproxy.io/docs/envoy/latest/api-v2/api/v2/core/base.proto#envoy-api-msg-core-node).
- Currently supported Node field types and identifiers:
  ```
  id = 0
  cluster = 1
  locality.region = 2
  locality.zone = 3;
  locality.subzone = 4;
  ```
- Ex:
  ```
  request_node_fragment:
  field: 0
  action:
    regex_action: { pattern: "^(.*)-*$", replace: "$1" }
  ```

##### `resource_names_fragment`
- Rules for generating the resulting fragment from xDS DiscoveryRequest resource names.
- _Note:_ We currently only support single resource elements.
- Ex:
  ```
  resource_names_fragment:
    element: 0
    action: { exact: true }
  ```

##### `string_fragment`
- A simple string fragment.
- Ex:
  ```
  string_fragment: lds
  ```

##### `and_result`
- A set that describes a logical AND. The result is a non-separated append operation between two or
  more fragments.
