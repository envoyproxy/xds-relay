// +build end2end,docker

package integration

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"syscall"
	"testing"
	"time"

	cachev2 "github.com/envoyproxy/go-control-plane/pkg/cache/v2"
	serverv2 "github.com/envoyproxy/go-control-plane/pkg/server/v2"
	serverv3 "github.com/envoyproxy/go-control-plane/pkg/server/v3"
	testgcp "github.com/envoyproxy/go-control-plane/pkg/test"
	resourcev2 "github.com/envoyproxy/go-control-plane/pkg/test/resource/v2"
	testgcpv2 "github.com/envoyproxy/go-control-plane/pkg/test/v2"
	"github.com/onsi/gomega"
)

func TestMain(m *testing.M) {
	// We force a 1 second sleep before running a test to let the OS close any lingering socket from previous
	// tests.
	time.Sleep(1 * time.Second)
	code := m.Run()
	os.Exit(code)
}

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
	cbv2 := &testgcpv2.Callbacks{Signal: signal}

	configv2 := cachev2.NewSnapshotCache(false, cachev2.IDHash{}, logger{})
	srv2 := serverv2.NewServer(ctx, configv2, cbv2)
	// TODO: do we have to initialize unused_srv3?
	unused_srv3 := serverv3.NewServer(ctx, nil, nil)

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
	go testgcp.RunManagementServer(ctx, srv2, unused_srv3, port)

	// TODO: parametrize bootstrap file
	envoyCmd := exec.CommandContext(ctx, "envoy", "-c", "./testdata/bootstrap.yaml", "--log-level", "debug")
	var b bytes.Buffer
	envoyCmd.Stdout = &b
	envoyCmd.Stderr = &b
	// Golang does not offer a portable solution to kill all child processes upon parent exit, so we rely on
	// this linuxism to send a SIGKILL to the envoy process (and its child sub-processes) when the parent (the
	// test) exits. More information in http://man7.org/linux/man-pages/man2/prctl.2.html
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

		g.Eventually(func() (int, int) {
			ok, failed := callLocalService(basePort, nHttpListeners, nTcpListeners)
			log.Printf("Request batch: ok %v, failed %v\n", ok, failed)
			return ok, failed
		}, 1*time.Second, 100*time.Millisecond).Should(gomega.Equal(nHttpListeners + nTcpListeners))

		cbv2.Report()
	}

	// TODO: figure out a way to only only copy envoy logs in case of failures. Maybe
	// use the github action for copying artifacts.
	// defer log.Printf("Envoy logs: \n%s", b.String())
}

func TestSnapshotCacheAndSingleEnvoy(t *testing.T) {
	test(t)
}

func TestSnapshotCacheAndSingleEnvoy2(t *testing.T) {
	test(t)
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

type logger struct{}

func (logger logger) Debugf(format string, args ...interface{}) {
	log.Printf(format+"\n", args...)
}

func (logger logger) Infof(format string, args ...interface{}) {
	log.Printf(format+"\n", args...)
}

func (logger logger) Warnf(format string, args ...interface{}) {
	log.Printf(format+"\n", args...)
}

func (logger logger) Errorf(format string, args ...interface{}) {
	log.Printf(format+"\n", args...)
}
