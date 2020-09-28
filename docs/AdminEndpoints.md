# Admin Endpoints

## `GET /cache/<key>`
### Behavior: 
Searches through xds-relay cache and outputs contents of the cached xDS response for the given aggregated key.
### Parameters:
*`<key>`*: The aggregated key or prefix to search
  * Argument supports wildcard (*) glob suffix patterns
  * Only supports single wildcard at the end of input key (e.g. `/cache/a*c*` is not supported)
  * `/cache`, `/cache/`, and `/cache/*` returns the entire cache
### Example: 
    * Keys in cache: abc_1, abc_2, abz_1, abz_2
    * Request: /cache/abc*
    * Output: Entries for abc_1 and abc_2

## `POST /log_level/<level>`
### Behavior:
Updates xds-relay log level
### Parameters:
*`<level>`*: Desired log level (supports `info`, `debug`, `warn`, or `error` levels)
* if no `<level>` provided, the aggregator outputs the current log level
### Example: 
    * Log level before update: debug
    * Request: /log_level/info
    * Output: "Current log level: info"

## `GET /server_info`
### Behavior:
Outputs server configuration (i.e. bootstrap configuration)
### Parameters:
n/a
### Example: 
    * xds-relay server's bootstrap file: ../example/config-files/xds-relay-bootstrap.yaml
    * Request: `/server_info`
    * Output (shortened to keep example brief, but outputs all bootstrap configurations): 
    ```json
    {
      "server": {
        "address": {
          "address": "0.0.0.0",
          "port_value": 9991
        }
      },
      "origin_server": {...},
      "logging": {...},
      "cache": {...},
      "metrics_sink": {...},
      "admin": {...}
    }
    ```

    