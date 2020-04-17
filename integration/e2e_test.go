// +build integrationdocker

package integration

import (
	"bytes"
	"context"
	"log"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSetupEnvoy(t *testing.T) {
	ctx := context.Background()

	log.Println("wow")

	envoyCmd := exec.CommandContext(ctx, "envoy", "-c", "./testdata/bootstrap.yaml", "--log-level", "debug")
	var b bytes.Buffer
	envoyCmd.Stdout = &b
	envoyCmd.Stderr = &b
	envoyCmd.Start()

	time.Sleep(time.Second * 1)

	defer log.Printf("Envoy logs: \n%s", b.String())

	assert.Equal(t, 1, 1)
}
