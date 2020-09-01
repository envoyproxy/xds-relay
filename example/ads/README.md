<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [ADS Example](#ads-example)
  - [Setup control plane](#setup-control-plane)
    - [Local setup from scratch](#local-setup-from-scratch)
  - [Run xds-relay server](#run-xds-relay-server)
  - [Run envoy to connect to xds-relay](#run-envoy-to-connect-to-xds-relay)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## ADS Example

### Setup control plane

#### Local setup from scratch

Use [kind](https://github.com/kubernetes-sigs/kind) to create a local k8s cluster

```
kind create cluster
```

Install istio

```
istioctl  install --set profile=minimal
```

Forward local 15010 to istiod 15010

```
kubectl -n istio-system port-forward --address 0.0.0.0 pod/${ISTIOD_POD_NAME} 15010:15010
```

Update upstream server address in `example/ads/config.yaml`

### Run xds-relay server

```
make compile

./bin/xds-relay -c example/ads/config.yaml -a example/ads/rules.yaml
```

### Run envoy to connect to xds-relay

Update xds-relay server address in `example/ads/front-envoy.yaml`

Start envoy

```
cd example/ads && docker-compose up --build
```

Verify config

```
curl http://127.0.0.1:8001/config_dump
```