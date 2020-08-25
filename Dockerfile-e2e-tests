FROM envoyproxy/envoy:v1.14.4 AS envoyproxy
FROM golang:1.14.6

WORKDIR /xds-relay

# add envoy
COPY --from=envoyproxy /usr/local/bin/envoy /usr/local/bin/envoy
