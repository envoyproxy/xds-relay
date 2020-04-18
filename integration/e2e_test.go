// +build end2end

package integration

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"runtime"
	"syscall"
	"testing"
	"time"

	cachev2 "github.com/envoyproxy/go-control-plane/pkg/cache/v2"
	cachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	serverv2 "github.com/envoyproxy/go-control-plane/pkg/server/v2"
	serverv3 "github.com/envoyproxy/go-control-plane/pkg/server/v3"
	testgcp "github.com/envoyproxy/go-control-plane/pkg/test"
	resourcev2 "github.com/envoyproxy/go-control-plane/pkg/test/resource/v2"
	testgcpv2 "github.com/envoyproxy/go-control-plane/pkg/test/v2"
	testgcpv3 "github.com/envoyproxy/go-control-plane/pkg/test/v3"
	"github.com/onsi/gomega"
)

func test(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Golang does not offer a cross-platform safe way of killing child processes, so we skip these tests if not on linux.")
	}
	g := gomega.NewWithT(t)

	ctx, cancelFunc := context.WithCancel(context.Background())
	// We cancel the context to make sure that all resources are cleaned on test completion
	defer func() {
		// TODO: confirm all resources were deallocated?
		log.Printf("Called at the end?")
		cancelFunc()
	}()

	debug := true
	var port uint = 18000
	var upstreamPort uint = 18080
	var basePort uint = 9000
	nClusters := 3
	nHttpListeners := 2
	nTcpListeners := 3
	nUpdates := 2

	// We run a service that returns the string "Hi, there!" locally and expose it through
	// envoy.
	go testgcp.RunHTTP(ctx, upstreamPort)

	// Create a cache
	signal := make(chan struct{})
	cbv2 := &testgcpv2.Callbacks{Signal: signal, Debug: debug}
	cbv3 := &testgcpv3.Callbacks{Signal: signal, Debug: debug}

	configv2 := cachev2.NewSnapshotCache(false, cachev2.IDHash{}, logger{Debug: debug})
	configv3 := cachev3.NewSnapshotCache(false, cachev3.IDHash{}, logger{Debug: debug})
	srv2 := serverv2.NewServer(ctx, configv2, cbv2)
	// TODO: do we have to initialize srv3?
	srv3 := serverv3.NewServer(ctx, configv3, cbv3)

	// Create a test snapshot
	snapshotv2 := resourcev2.TestSnapshot{
		Xds:              "xds",
		UpstreamPort:     uint32(upstreamPort),
		BasePort:         uint32(basePort),
		NumClusters:      nClusters,
		NumHTTPListeners: nHttpListeners,
		NumTCPListeners:  nTcpListeners,
	}

	// Start the xDS server
	go testgcp.RunManagementServer(ctx, srv2, srv3, port)

	// TODO: parametrize bootstrap file
	envoyCmd := exec.CommandContext(ctx, "envoy", "-c", "./testdata/bootstrap.yaml", "--log-level", "debug")
	var b bytes.Buffer
	envoyCmd.Stdout = &b
	envoyCmd.Stderr = &b
	envoyCmd.SysProcAttr = &syscall.SysProcAttr{Pdeathsig: syscall.SIGKILL}
	envoyCmd.Start()

	log.Println("Waiting for the first request...")
	select {
	case <-signal:
		break
	case <-time.After(1 * time.Minute):
		log.Printf("Envoy logs: \n%s", b.String())
		t.Fatalf("Timeout waiting for the first request")
	}

	for i := 0; i < nUpdates; i++ {
		snapshotv2.Version = fmt.Sprintf("v%d", i)
		log.Printf("Update snapshot %v\n", snapshotv2.Version)

		snapshotv2 := snapshotv2.Generate()
		if err := snapshotv2.Consistent(); err != nil {
			log.Printf("Snapshot inconsistency: %+v\n", snapshotv2)
		}

		// TODO: parametrize node-id in bootstrap files. Maybe adopt templates?
		err := configv2.SetSnapshot("test-id", snapshotv2)
		if err != nil {
			t.Fatalf("Snapshot error %q for %+v\n", err, snapshotv2)
		}

		pass := false
		for j := 0; j < nRequests; j++ {
			ok, failed := callLocalService(basePort, nHttpListeners, nTcpListeners)
			if failed == 0 && !pass {
				pass = true
			}
			log.Printf("request batch %d, ok %v, failed %v, pass %v\n", j, ok, failed, pass)

			select {
			case <-time.After(1 * time.Second):
			case <-ctx.Done():
				return
			}
		}

		cbv2.Report()

		if !pass {
			log.Printf("Envoy logs: \n%s", b.String())
			t.Fatalf("Failed all requests in a run")
		}
	}

	assert.Equal(t, 1, 1)
}

func callLocalService(basePort uint, nHttpListeners int, nTcpListeners int) (int, int) {
	total := nHttpListeners + nTcpListeners
	ok, failed := 0, 0
	ch := make(chan error, total)

	// spawn requests
	for i := 0; i < total; i++ {
		go func(i int) {
			client := http.Client{
				Timeout:   100 * time.Millisecond,
				Transport: &http.Transport{},
			}
			req, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d", basePort+uint(i)))
			if err != nil {
				ch <- err
				return
			}
			defer req.Body.Close()
			body, err := ioutil.ReadAll(req.Body)
			if err != nil {
				ch <- err
				return
			}
			if string(body) != testgcp.Hello {
				ch <- fmt.Errorf("unexpected return %q", string(body))
				return
			}
			ch <- nil
		}(i)
	}

	for {
		out := <-ch
		if out == nil {
			ok++
		} else {
			failed++
		}
		if ok+failed == total {
			return ok, failed
		}
	}
}

type logger struct {
	Debug bool
}

func (logger logger) Debugf(format string, args ...interface{}) {
	if logger.Debug {
		log.Printf(format+"\n", args...)
	}
}

func (logger logger) Infof(format string, args ...interface{}) {
	if logger.Debug {
		log.Printf(format+"\n", args...)
	}
}

func (logger logger) Warnf(format string, args ...interface{}) {
	log.Printf(format+"\n", args...)
}

func (logger logger) Errorf(format string, args ...interface{}) {
	log.Printf(format+"\n", args...)
}
