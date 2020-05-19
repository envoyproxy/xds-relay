// +build integration

package integration

import (
	"os"
	"os/exec"
	"path"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const (
	binaryName = "xds-relay"
)

func TestServerShutdown(t *testing.T) {
	dir, err := os.Getwd()
	assert.Nil(t, err)
	cmd := exec.Command(path.Join(dir, "bin", binaryName),
		"-c", "./integration/testdata/bootstrap_configuration_complete_tech_spec.yaml",
		"-a", "./integration/testdata/keyer_configuration_complete_tech_spec.yaml",
		"-l", "debug",
		"-m", "serve")
	err = cmd.Start()
	assert.Nil(t, err)
	<-time.After(5 * time.Second)
	assert.Equal(t, -1, cmd.ProcessState.ExitCode())
	e := cmd.Process.Signal(syscall.SIGINT)
	assert.Nil(t, e)
	err = cmd.Wait()
	assert.Nil(t, e)
}

func TestAssertAlwaysFalse(t *testing.T) {
	assert.Equal(t, 1, 2)
}
