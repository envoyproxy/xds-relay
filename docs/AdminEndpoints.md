# Admin Endpoints

## `GET /cache/<key>`
Behavior: searches through xDS Aggregator cache and outputs contents for argument `<key>`

* `<key>` argument supports wildcard (*) glob suffix patterns 
* `/cache`, `/cache/`, and `/cache/*` returns the entire cache
* Example: 
    * Request: `/cache/abc*`
    * Keys in cache: `abc_1`, `abc_2`, `abz_1`, `abz_2`
    * Output: Entries for `abc_1` and `abc_2`
* Endpoint only supports single wildcard (*) at end of input key (`a*c*` is not supported)


## `POST /log_level/<level>`
Behavior: Changes xDS Aggregator's log level to desired `<level>`

* `<level>` argument is mandatory and supports DEBUG, WARN, or ERROR
* if no `<level>` provided, the aggregator outputs the current log level


## `GET /server_info`
Behavior: Outputs server configuration (i.e. bootstrap configuration)
