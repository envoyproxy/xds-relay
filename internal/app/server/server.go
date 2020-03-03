// Copyright 2020 Envoyproxy Authors
//
//   Licensed under the Apache License, Version 2.0 (the "License");
//   you may not use this file except in compliance with the License.
//   You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
//   Unless required by applicable law or agreed to in writing, software
//   distributed under the License is distributed on an "AS IS" BASIS,
//   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//   See the License for the specific language governing permissions and
//   limitations under the License.

package server

import (
	"context"
	"net"

	"github.com/envoyproxy/xds-relay/internal/pkg/log"

	api "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/envoyproxy/go-control-plane/pkg/cache"
	gcp "github.com/envoyproxy/go-control-plane/pkg/server"
	"google.golang.org/grpc"
)

// Run instantiates a running gRPC server for accepting incoming xDS-based
// requests.
func Run() {
	logger := log.New().Sugar()

	// Cursory implementation of go-control-plane's server.
	// TODO cancel should be invoked by shutdown handlers.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	snapshotCache := cache.NewSnapshotCache(false, cache.IDHash{}, nil)
	gcpServer := gcp.NewServer(ctx, snapshotCache, nil)
	server := grpc.NewServer()
	listener, err := net.Listen("tcp", ":8080") // #nosec
	if err != nil {
		logger.Fatalw("failed to bind server to listener", "err", err)
	}

	api.RegisterEndpointDiscoveryServiceServer(server, gcpServer)
	api.RegisterClusterDiscoveryServiceServer(server, gcpServer)
	api.RegisterRouteDiscoveryServiceServer(server, gcpServer)
	api.RegisterListenerDiscoveryServiceServer(server, gcpServer)

	logger.Info("Initializing server at", listener.Addr().String())
	if err := server.Serve(listener); err != nil {
		logger.Fatalw("failed to initialize server", "err", err)
	}
}
