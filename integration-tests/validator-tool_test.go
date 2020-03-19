package integration_tests

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"testing"
)

var binaryName = "configuration-validator"

func TestMain(m *testing.M) {
	// TODO I could not find a more robust way of reaching the root of the repo
	err := os.Chdir("..")
	if err != nil {
		fmt.Printf("could not change dir: %v", err)
		os.Exit(1)
	}

	make := exec.Command("make", "compile-validator-tool")
	err = make.Run()
	if err != nil {
		fmt.Printf("could not make binary for %s: %v", binaryName, err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func TestConfigurationValidatorTool(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"valid file", []string{"-config", "t.yaml"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}

			cmd := exec.Command(path.Join(dir, "bin", binaryName), tt.args...)
			fmt.Println(cmd)
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatal(string(output))
				t.Fatal(err)
			}

			fmt.Println(output)
		})
	}
}
