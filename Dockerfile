FROM golang:1.14.1

WORKDIR /xds-relay

# add envoy
COPY --from=envoyproxy/envoy:v1.13.1 /usr/local/bin/envoy /usr/local/bin/envoy
