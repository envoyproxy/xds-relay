package upstream

import (
	"context"

	"github.com/uber-go/tally"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	clusterservice "github.com/envoyproxy/go-control-plane/envoy/service/cluster/v3"
	discoveryv3 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	endpointservice "github.com/envoyproxy/go-control-plane/envoy/service/endpoint/v3"
	listenerservice "github.com/envoyproxy/go-control-plane/envoy/service/listener/v3"
	routeservice "github.com/envoyproxy/go-control-plane/envoy/service/route/v3"
	"github.com/envoyproxy/xds-relay/internal/pkg/log"
)

// NewMockClient creates a mock implementation for testing
func NewMockClient(
	ctx context.Context,
	ldsClient v2.ListenerDiscoveryServiceClient,
	rdsClient v2.RouteDiscoveryServiceClient,
	edsClient v2.EndpointDiscoveryServiceClient,
	cdsClient v2.ClusterDiscoveryServiceClient,
	callOptions CallOptions,
	scope tally.Scope,
	timeout int64) Client {
	return &client{
		ldsClient:   ldsClient,
		rdsClient:   rdsClient,
		edsClient:   edsClient,
		cdsClient:   cdsClient,
		callOptions: callOptions,
		logger:      log.MockLogger,
		scope:       scope,
		shutdown:    make(<-chan struct{}),
		timeout:     timeout,
	}
}

// NewMockClientV3 creates a mock implementation for testing
func NewMockClientV3(
	ctx context.Context,
	ldsClient listenerservice.ListenerDiscoveryServiceClient,
	rdsClient routeservice.RouteDiscoveryServiceClient,
	edsClient endpointservice.EndpointDiscoveryServiceClient,
	cdsClient clusterservice.ClusterDiscoveryServiceClient,
	callOptions CallOptions,
	scope tally.Scope,
	timeout int64) Client {
	return &client{
		ldsClientV3: ldsClient,
		rdsClientV3: rdsClient,
		edsClientV3: edsClient,
		cdsClientV3: cdsClient,
		callOptions: callOptions,
		logger:      log.MockLogger,
		scope:       scope,
		shutdown:    make(<-chan struct{}),
		timeout:     timeout,
	}
}

// NewMockClientEDS creates a mock implementation for testing both v2 and v3 eds together
func NewMockClientEDS(
	ctx context.Context,
	edsClientV3 endpointservice.EndpointDiscoveryServiceClient,
	edsClientV2 v2.EndpointDiscoveryServiceClient,
	callOptions CallOptions,
	scope tally.Scope) Client {
	return &client{
		edsClient:   edsClientV2,
		edsClientV3: edsClientV3,
		callOptions: callOptions,
		logger:      log.MockLogger,
		scope:       scope,
		shutdown:    make(<-chan struct{}),
	}
}

// NewMock creates a mock client implementation for testing
func NewMock(
	ctx context.Context,
	callOptions CallOptions,
	errorOnCreate []error,
	ldsReceiveChan chan *v2.DiscoveryResponse,
	rdsReceiveChan chan *v2.DiscoveryResponse,
	edsReceiveChan chan *v2.DiscoveryResponse,
	cdsReceiveChan chan *v2.DiscoveryResponse,
	sendCb func(m interface{}) error,
	scope tally.Scope,
	timeout int64) Client {
	return NewMockClient(
		ctx,
		createMockLdsClient(errorOnCreate, ldsReceiveChan, sendCb),
		createMockRdsClient(errorOnCreate, rdsReceiveChan, sendCb),
		createMockEdsClient(errorOnCreate, edsReceiveChan, sendCb),
		createMockCdsClient(errorOnCreate, cdsReceiveChan, sendCb),
		callOptions,
		scope,
		timeout,
	)
}

// NewMockV3 creates a mock client implementation for testing
func NewMockV3(
	ctx context.Context,
	callOptions CallOptions,
	errorOnCreate []error,
	ldsReceiveChan chan *discoveryv3.DiscoveryResponse,
	rdsReceiveChan chan *discoveryv3.DiscoveryResponse,
	edsReceiveChan chan *discoveryv3.DiscoveryResponse,
	cdsReceiveChan chan *discoveryv3.DiscoveryResponse,
	sendCb func(m interface{}) error,
	scope tally.Scope,
	timeout int64) Client {
	return NewMockClientV3(
		ctx,
		createMockLdsClientV3(errorOnCreate, ldsReceiveChan, sendCb),
		createMockRdsClientV3(errorOnCreate, rdsReceiveChan, sendCb),
		createMockEdsClientV3(errorOnCreate, edsReceiveChan, sendCb),
		createMockCdsClientV3(errorOnCreate, cdsReceiveChan, sendCb),
		callOptions,
		scope,
		timeout,
	)
}

// NewMockEDS creates a mock client implementation for testing v2 and v3 eds together
func NewMockEDS(
	ctx context.Context,
	callOptions CallOptions,
	errorOnCreate []error,
	edsReceiveChanV3 chan *discoveryv3.DiscoveryResponse,
	edsReceiveChanV2 chan *v2.DiscoveryResponse,
	sendCb func(m interface{}) error,
	scope tally.Scope) Client {
	return NewMockClientEDS(
		ctx,
		createMockEdsClientV3(errorOnCreate, edsReceiveChanV3, sendCb),
		createMockEdsClient(errorOnCreate, edsReceiveChanV2, sendCb),
		callOptions,
		scope,
	)
}

func createMockLdsClient(
	errorOnCreate []error,
	receiveChan chan *v2.DiscoveryResponse,
	sendCb func(m interface{}) error) v2.ListenerDiscoveryServiceClient {
	return &mockClient{errorOnStreamCreate: errorOnCreate, receiveChan: receiveChan, sendCb: sendCb}
}

func createMockLdsClientV3(
	errorOnCreate []error,
	receiveChan chan *discoveryv3.DiscoveryResponse,
	sendCb func(m interface{}) error) listenerservice.ListenerDiscoveryServiceClient {
	return &mockClientV3{errorOnStreamCreate: errorOnCreate, receiveChan: receiveChan, sendCb: sendCb}
}

func createMockCdsClient(
	errorOnCreate []error,
	receiveChan chan *v2.DiscoveryResponse,
	sendCb func(m interface{}) error) v2.ClusterDiscoveryServiceClient {
	return &mockClient{errorOnStreamCreate: errorOnCreate, receiveChan: receiveChan, sendCb: sendCb}
}

func createMockCdsClientV3(
	errorOnCreate []error,
	receiveChan chan *discoveryv3.DiscoveryResponse,
	sendCb func(m interface{}) error) clusterservice.ClusterDiscoveryServiceClient {
	return &mockClientV3{errorOnStreamCreate: errorOnCreate, receiveChan: receiveChan, sendCb: sendCb}
}

func createMockRdsClient(
	errorOnCreate []error,
	receiveChan chan *v2.DiscoveryResponse,
	sendCb func(m interface{}) error) v2.RouteDiscoveryServiceClient {
	return &mockClient{errorOnStreamCreate: errorOnCreate, receiveChan: receiveChan, sendCb: sendCb}
}

func createMockRdsClientV3(
	errorOnCreate []error,
	receiveChan chan *discoveryv3.DiscoveryResponse,
	sendCb func(m interface{}) error) routeservice.RouteDiscoveryServiceClient {
	return &mockClientV3{errorOnStreamCreate: errorOnCreate, receiveChan: receiveChan, sendCb: sendCb}
}

func createMockEdsClient(
	errorOnCreate []error,
	receiveChan chan *v2.DiscoveryResponse,
	sendCb func(m interface{}) error) v2.EndpointDiscoveryServiceClient {
	return &mockClient{errorOnStreamCreate: errorOnCreate, receiveChan: receiveChan, sendCb: sendCb}
}

func createMockEdsClientV3(
	errorOnCreate []error,
	receiveChan chan *discoveryv3.DiscoveryResponse,
	sendCb func(m interface{}) error) endpointservice.EndpointDiscoveryServiceClient {
	return &mockClientV3{errorOnStreamCreate: errorOnCreate, receiveChan: receiveChan, sendCb: sendCb}
}
