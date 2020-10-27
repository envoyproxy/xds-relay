// +build end2end,docker

package integration

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/envoyproxy/xds-relay/internal/pkg/log"

	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	gcpcachev2 "github.com/envoyproxy/go-control-plane/pkg/cache/v2"
	gcpcachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	gcpserverv2 "github.com/envoyproxy/go-control-plane/pkg/server/v2"
	gcpserverv3 "github.com/envoyproxy/go-control-plane/pkg/server/v3"
	gcptest "github.com/envoyproxy/go-control-plane/pkg/test"
	gcpresourcev2 "github.com/envoyproxy/go-control-plane/pkg/test/resource/v2"
	gcpresourcev3 "github.com/envoyproxy/go-control-plane/pkg/test/resource/v3"
	gcptestv2 "github.com/envoyproxy/go-control-plane/pkg/test/v2"
	gcptestv3 "github.com/envoyproxy/go-control-plane/pkg/test/v3"
	"github.com/envoyproxy/xds-relay/internal/app/server"
	yamlproto "github.com/envoyproxy/xds-relay/internal/pkg/util/yamlproto"
	aggregationv1 "github.com/envoyproxy/xds-relay/pkg/api/aggregation/v1"
	bootstrapv1 "github.com/envoyproxy/xds-relay/pkg/api/bootstrap/v1"
	"github.com/onsi/gomega"
)

var testLogger = log.MockLogger.Named("e2e")

// Test parameters
const (
	managementServerPort uint = 18000 // gRPC management server port
	httpServicePort      uint = 18080 // upstream HTTP/1.1 service that Envoy wll call
	envoyListenerPort    uint = 9000  // initial port for the Envoy listeners generated by the snapshot cache
	nClusters                 = 7
	nListeners                = 9
	nUpdates                  = 4
	keyerConfiguration        = "./testdata/keyer_configuration_e2e.yaml"
	xdsRelayBootstrap         = "./testdata/bootstrap_configuration_e2e.yaml"
	upstreamMessage           = "Hi, there!"
)

func TestMain(m *testing.M) {
	// We force a 1 second sleep before running a test to let the OS close any lingering socket from previous
	// tests.
	time.Sleep(1 * time.Second)
	code := m.Run()
	os.Exit(code)
}

func TestSnapshotCacheSingleEnvoyAndXdsRelayServer(t *testing.T) {
	g := gomega.NewWithT(t)
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	// We run a service that returns the string "Hi, there!" locally and expose it through envoy.
	// This is the service that Envoy will make requests to.
	go runUpstream(httpServicePort)

	// Mimic a management server using go-control-plane's snapshot cache.
	configv2, configv3, signalv2, signalv3 := startSnapshotCache(ctx, managementServerPort)

	// Start xds-relay server.
	startXdsRelayServer(ctx, cancelFunc, xdsRelayBootstrap, keyerConfiguration)

	for _, version := range []core.ApiVersion{core.ApiVersion_V2, core.ApiVersion_V3} {
		t.Run(version.String(), func(t *testing.T) {
			// Start envoy and return a bytes buffer containing the envoy logs.
			var signal chan struct{}
			if version == core.ApiVersion_V2 {
				signal = signalv2
			} else {
				signal = signalv3
			}
			envoyLogsBuffer, pid := startEnvoy(ctx, getEnvoyBootstrap(version), signal)

			for i := 0; i < nUpdates; i++ {
				setSnapshot(ctx, i, version, configv2, configv3)

				g.Eventually(func() (int, int) {
					ok, failed := callLocalService(envoyListenerPort, nListeners)
					testLogger.Info(ctx, "asserting envoy listeners configured: ok %v, failed %v", ok, failed)
					return ok, failed
				}, 1*time.Second, 100*time.Millisecond).Should(gomega.Equal(nListeners))
			}

			// If envoy never starts, the buffer is empty
			assert.NotEmpty(t, envoyLogsBuffer.String())
			// TODO(https://github.com/envoyproxy/xds-relay/issues/66): figure out a way to only only copy
			// envoy logs in case of failures.
			testLogger.With("envoy_logs", envoyLogsBuffer.String()).Debug(ctx, "captured envoy logs")

			err := stopEnvoy(ctx, pid)
			assert.NoError(t, err)
		})
	}
}

func setSnapshot(
	ctx context.Context,
	updateIndex int,
	xdsVersion core.ApiVersion,
	configv2 gcpcachev2.SnapshotCache,
	configv3 gcpcachev3.SnapshotCache) {
	switch xdsVersion {
	case core.ApiVersion_V2:
		snapshotConfig := gcpresourcev2.TestSnapshot{
			Xds:              "xds",
			UpstreamPort:     uint32(httpServicePort),
			BasePort:         uint32(envoyListenerPort),
			NumClusters:      nClusters,
			NumHTTPListeners: nListeners,
		}
		// Bumping the snapshot version mimics new management server configuration.
		snapshotConfig.Version = fmt.Sprintf("v%d", updateIndex)
		testLogger.Info(ctx, "updating snapshots to version: %v", snapshotConfig.Version)

		snapshot := snapshotConfig.Generate()
		if err := snapshot.Consistent(); err != nil {
			testLogger.Fatal(ctx, "snapshot inconsistency: %+v", snapshot)
		}
		err := configv2.SetSnapshot("envoy-1", snapshot)
		if err != nil {
			testLogger.Fatal(ctx, "set snapshot error %q for %+v", err, snapshot)
		}
	case core.ApiVersion_V3:
		snapshotConfig := gcpresourcev3.TestSnapshot{
			Xds:              "xds",
			UpstreamPort:     uint32(httpServicePort),
			BasePort:         uint32(envoyListenerPort),
			NumClusters:      nClusters,
			NumHTTPListeners: nListeners,
		}
		// Bumping the snapshot version mimics new management server configuration.
		snapshotConfig.Version = fmt.Sprintf("v%d", updateIndex)
		testLogger.Info(ctx, "updating snapshots to version: %v", snapshotConfig.Version)

		snapshot := snapshotConfig.Generate()
		if err := snapshot.Consistent(); err != nil {
			testLogger.Fatal(ctx, "snapshot inconsistency: %+v", snapshot)
		}
		err := configv3.SetSnapshot("envoy-1", snapshot)
		if err != nil {
			testLogger.Fatal(ctx, "set snapshot error %q for %+v", err, snapshot)
		}
	}
}

func getEnvoyBootstrap(version core.ApiVersion) string {
	switch version {
	case core.ApiVersion_V2:
		return "./testdata/envoy_bootstrap.yaml"
	case core.ApiVersion_V3:
		return "./testdata/envoy_bootstrap_v3.yaml"
	default:
		return ""
	}
}

func startSnapshotCache(
	ctx context.Context,
	port uint) (gcpcachev2.SnapshotCache, gcpcachev3.SnapshotCache, chan struct{}, chan struct{}) {
	// Create a cache
	signalv2 := make(chan struct{})
	signalv3 := make(chan struct{})
	cbv2 := &gcptestv2.Callbacks{Signal: signalv2}
	cbv3 := &gcptestv3.Callbacks{Signal: signalv3}
	configv2 := gcpcachev2.NewSnapshotCache(false, gcpcachev2.IDHash{}, gcpLogger{logger: testLogger.Named("snapshotv2")})
	configv3 := gcpcachev3.NewSnapshotCache(false, gcpcachev3.IDHash{}, gcpLogger{logger: testLogger.Named("snapshotv3")})
	srv2 := gcpserverv2.NewServer(ctx, configv2, cbv2)
	srv3 := gcpserverv3.NewServer(ctx, configv3, cbv3)

	// Start up a gRPC-based management server.
	go gcptest.RunManagementServer(ctx, srv2, srv3, port)

	return configv2, configv3, signalv2, signalv3
}

func startXdsRelayServer(ctx context.Context, cancel context.CancelFunc, bootstrapConfigFilePath string,
	keyerConfigurationFilePath string) {
	bootstrapConfigFileContent, err := ioutil.ReadFile(bootstrapConfigFilePath)
	if err != nil {
		testLogger.Fatal(ctx, "failed to read bootstrap config file: ", err)
	}
	var bootstrapConfig bootstrapv1.Bootstrap
	err = yamlproto.FromYAMLToBootstrapConfiguration(string(bootstrapConfigFileContent), &bootstrapConfig)
	if err != nil {
		testLogger.Fatal(ctx, "failed to translate bootstrap config: ", err)
	}

	aggregationRulesFileContent, err := ioutil.ReadFile(keyerConfigurationFilePath)
	if err != nil {
		testLogger.Fatal(ctx, "failed to read aggregation rules file: ", err)
	}
	var aggregationRulesConfig aggregationv1.KeyerConfiguration
	err = yamlproto.FromYAMLToKeyerConfiguration(string(aggregationRulesFileContent), &aggregationRulesConfig)
	if err != nil {
		testLogger.Fatal(ctx, "failed to translate aggregation rules: ", err)
	}
	go server.RunWithContext(ctx, cancel, &bootstrapConfig, &aggregationRulesConfig, "debug", "serve")
}

func startEnvoy(ctx context.Context, bootstrapFilePath string, signal chan struct{}) (bytes.Buffer, int) {
	envoyCmd := exec.CommandContext(ctx, "envoy", "-c", bootstrapFilePath, "--log-level", "debug")
	var b bytes.Buffer
	envoyCmd.Stdout = &b
	envoyCmd.Stderr = &b
	envoyCmd.Start()

	testLogger.Info(ctx, "waiting for upstream cluster to send the first response ...")
	select {
	case <-signal:
		break
	case <-time.After(1 * time.Minute):
		testLogger.Info(ctx, "envoy logs: \n%s", b.String())
		testLogger.Fatal(ctx, "timeout waiting for upstream cluster to send the first response")
	}

	return b, envoyCmd.Process.Pid
}

func stopEnvoy(ctx context.Context, pid int) error {
	killCmd := exec.CommandContext(ctx, "kill", "-9", strconv.Itoa(pid))
	err := killCmd.Start()
	if err != nil {
		return err
	}
	return killCmd.Wait()
}

func callLocalService(port uint, nListeners int) (int, int) {
	ok, failed := 0, 0
	ch := make(chan error, nListeners)

	// spawn requests
	for i := 0; i < nListeners; i++ {
		go func(i int) {
			client := http.Client{
				Timeout:   100 * time.Millisecond,
				Transport: &http.Transport{},
			}
			req, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d", port+uint(i)))
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
			if string(body) != upstreamMessage {
				ch <- fmt.Errorf("expected envoy response: %q, got: %q", upstreamMessage, string(body))
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
			testLogger.With("error", out).Info(context.Background(), "envoy request error")
			failed++
		}
		if ok+failed == nListeners {
			return ok, failed
		}
	}
}

func runUpstream(upstreamPort uint) {
	testLogger.Info(context.Background(), "upstream listening HTTP/1.1 on %d\n", upstreamPort)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write([]byte(upstreamMessage)); err != nil {
			testLogger.Info(context.Background(), err.Error())
		}
	})
	if err := http.ListenAndServe(fmt.Sprintf(":%d", upstreamPort), nil); err != nil {
		testLogger.Info(context.Background(), err.Error())
	}
}
