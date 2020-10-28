package main

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"os"
	"time"

	"google.golang.org/grpc"

	api "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v2"
	xds "github.com/envoyproxy/go-control-plane/pkg/server/v2"
	gcpresourcev2 "github.com/envoyproxy/go-control-plane/pkg/test/resource/v2"
)

func generateTestSnapshotNewVersion(snapshotCache cache.SnapshotCache) {
	// infinite loop where we wait for some time and override objects in the cache

	snapshotConfig := gcpresourcev2.TestSnapshot{
		Xds:              "xds",
		UpstreamPort:     uint32(12000),
		BasePort:         uint32(9000),
		NumClusters:      3,
		NumHTTPListeners: 2,
	}

	// Add some randomness in the definition of the new version
	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)

	for i := 0; i < 100; i++ {
		fmt.Println("wait 10s before generating the new version")
		time.Sleep(10 * time.Second)

		newVersion := r1.Intn(100000)
		fmt.Printf("new version = v%d\n", newVersion)
		snapshotConfig.Version = fmt.Sprintf("v%d", newVersion)
		snapshot := snapshotConfig.Generate()
		if err := snapshot.Consistent(); err != nil {
			fmt.Printf("snapshot inconsistency: %+v", snapshot)
		}
		err := snapshotCache.SetSnapshot("envoy-client-1", snapshot)
		if err != nil {
			fmt.Printf("set snapshot error %q for %+v", err, snapshot)
		}
	}

	fmt.Println("reached the end of the xDS generation data. Exiting the program.")
	os.Exit(0)
}

func runServer(snapshotCache cache.SnapshotCache, port int) {
	server := xds.NewServer(context.Background(), snapshotCache, nil)
	grpcServer := grpc.NewServer()
	listener, _ := net.Listen("tcp", fmt.Sprintf(":%d", port))

	discovery.RegisterAggregatedDiscoveryServiceServer(grpcServer, server)
	api.RegisterEndpointDiscoveryServiceServer(grpcServer, server)
	api.RegisterClusterDiscoveryServiceServer(grpcServer, server)
	api.RegisterRouteDiscoveryServiceServer(grpcServer, server)
	api.RegisterListenerDiscoveryServiceServer(grpcServer, server)

	if err := grpcServer.Serve(listener); err != nil {
		fmt.Println("something went wrong in the server")
	}
}

func main() {
	managementServerPort := 18000

	snapshotCache := cache.NewSnapshotCache(false, cache.IDHash{}, nil)

	// Start producing new versions of the test snapshot cache in a goroutine
	go generateTestSnapshotNewVersion(snapshotCache)

	runServer(snapshotCache, managementServerPort)
}
