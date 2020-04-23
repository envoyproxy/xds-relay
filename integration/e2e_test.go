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

	gcpcachev2 "github.com/envoyproxy/go-control-plane/pkg/cache/v2"
	gcpserverv2 "github.com/envoyproxy/go-control-plane/pkg/server/v2"
	gcpserverv3 "github.com/envoyproxy/go-control-plane/pkg/server/v3"
	gcptest "github.com/envoyproxy/go-control-plane/pkg/test"
	gcpresourcev2 "github.com/envoyproxy/go-control-plane/pkg/test/resource/v2"
	gcptestv2 "github.com/envoyproxy/go-control-plane/pkg/test/v2"
	"github.com/envoyproxy/xds-relay/internal/app/server"
	yamlproto "github.com/envoyproxy/xds-relay/internal/pkg/util"
	aggregationv1 "github.com/envoyproxy/xds-relay/pkg/api/aggregation/v1"
	bootstrapv1 "github.com/envoyproxy/xds-relay/pkg/api/bootstrap/v1"
	"github.com/onsi/gomega"
)

func TestMain(m *testing.M) {
	// We force a 1 second sleep before running a test to let the OS close any lingering socket from previous
	// tests.
	time.Sleep(1 * time.Second)
	code := m.Run()
	os.Exit(code)
}

func TestSnapshotCacheSingleEnvoyAndXdsRelayServer(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Golang does not offer a cross-platform safe way of killing child processes, so we skip these tests if not on linux.")
	}

	g := gomega.NewWithT(t)

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	// Test parameters
	const (
		port               uint = 18000
		upstreamPort       uint = 18080
		basePort           uint = 9000
		nClusters               = 7
		nListeners              = 9
		nUpdates                = 4
		keyerConfiguration      = "./testdata/keyer_configuration_complete_tech_spec.yaml"
		xdsRelayBootstrap       = "./testdata/bootstrap_configuration_e2e.yaml"
		envoyBootstrap          = "./testdata/envoy_bootstrap.yaml"
	)

	// We run a service that returns the string "Hi, there!" locally and expose it through envoy.
	go gcptest.RunHTTP(ctx, upstreamPort)

	configv2, snapshotv2, signal := startSnapshotCache(ctx, upstreamPort, basePort, nClusters, nListeners, port)

	// Start xds-relay server. Note that we are starting the server now but the envoy instance is not yet
	// connecting to it since the orchestrator implementation is still a work in progress.
	startXdsRelayServer(ctx, xdsRelayBootstrap, keyerConfiguration)

	// Start envoy and return a bytes buffer containing the envoy logs
	// TODO: hook up envoy to the xds-relay server
	envoyLogsBuffer := startEnvoy(ctx, envoyBootstrap, signal)

	for i := 0; i < nUpdates; i++ {
		snapshotv2.Version = fmt.Sprintf("v%d", i)
		log.Printf("Update snapshot %v\n", snapshotv2.Version)

		snapshotv2 := snapshotv2.Generate()
		if err := snapshotv2.Consistent(); err != nil {
			log.Printf("Snapshot inconsistency: %+v\n", snapshotv2)
		}

		// TODO: parametrize node-id in bootstrap files.
		err := configv2.SetSnapshot("test-id", snapshotv2)
		if err != nil {
			t.Fatalf("Snapshot error %q for %+v\n", err, snapshotv2)
		}

		g.Eventually(func() (int, int) {
			ok, failed := callLocalService(basePort, nListeners)
			log.Printf("Request batch: ok %v, failed %v\n", ok, failed)
			return ok, failed
		}, 1*time.Second, 100*time.Millisecond).Should(gomega.Equal(nListeners))
	}

	// TODO: figure out a way to only only copy envoy logs in case of failures. Maybe
	// use the github action for copying artifacts (and not dump envoy logs to stdout).
	log.Printf("Envoy logs: \n%s", envoyLogsBuffer.String())
}

func startSnapshotCache(ctx context.Context, upstreamPort uint, basePort uint, nClusters int, nListeners int, port uint) (gcpcachev2.SnapshotCache, gcpresourcev2.TestSnapshot, chan struct{}) {
	// Create a cache
	signal := make(chan struct{})
	cbv2 := &gcptestv2.Callbacks{Signal: signal}

	configv2 := gcpcachev2.NewSnapshotCache(false, gcpcachev2.IDHash{}, logger{})
	srv2 := gcpserverv2.NewServer(ctx, configv2, cbv2)
	// TODO: do we have to initialize unused_srv3?
	unused_srv3 := gcpserverv3.NewServer(ctx, nil, nil)

	// Create a test snapshot
	snapshotv2 := gcpresourcev2.TestSnapshot{
		Xds:              "xds",
		UpstreamPort:     uint32(upstreamPort),
		BasePort:         uint32(basePort),
		NumClusters:      nClusters,
		NumHTTPListeners: nListeners,
	}

	// Start the xDS server
	go gcptest.RunManagementServer(ctx, srv2, unused_srv3, port)

	return configv2, snapshotv2, signal
}

func startXdsRelayServer(ctx context.Context, bootstrapConfigFilePath string, keyerConfigurationFilePath string) {
	bootstrapConfigFileContent, err := ioutil.ReadFile(bootstrapConfigFilePath)
	if err != nil {
		log.Fatal("failed to read bootstrap config file: ", err)
	}
	var bootstrapConfig bootstrapv1.Bootstrap
	err = yamlproto.FromYAMLToBootstrapConfiguration(string(bootstrapConfigFileContent), &bootstrapConfig)
	if err != nil {
		log.Fatal("failed to translate bootstrap config: ", err)
	}

	aggregationRulesFileContent, err := ioutil.ReadFile(keyerConfigurationFilePath)
	if err != nil {
		log.Fatal("failed to read aggregation rules file: ", err)
	}
	var aggregationRulesConfig aggregationv1.KeyerConfiguration
	err = yamlproto.FromYAMLToKeyerConfiguration(string(aggregationRulesFileContent), &aggregationRulesConfig)
	if err != nil {
		log.Fatal("failed to translate aggregation rules: ", err)
	}
	go server.RunWithContext(ctx, &bootstrapConfig, &aggregationRulesConfig, "debug", "serve")
}

func startEnvoy(ctx context.Context, bootstrapFilePath string, signal chan struct{}) bytes.Buffer {
	envoyCmd := exec.CommandContext(ctx, "envoy", "-c", bootstrapFilePath, "--log-level", "debug")
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
		log.Fatalf("Timeout waiting for the first request")
	}

	return b
}

func callLocalService(basePort uint, nListeners int) (int, int) {
	ok, failed := 0, 0
	ch := make(chan error, nListeners)

	// spawn requests
	for i := 0; i < nListeners; i++ {
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
			if string(body) != gcptest.Hello {
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
		if ok+failed == nListeners {
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
