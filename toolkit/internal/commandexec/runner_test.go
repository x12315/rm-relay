package commandexec

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOSRunnerPreservesExplicitEnvironmentAndWorkingDirectory(t *testing.T) {
	workingDirectory := t.TempDir()
	result, err := (OSRunner{}).Run(context.Background(), Request{
		Name:      os.Args[0],
		Arguments: []string{"-test.run=TestOSRunnerHelperProcess", "--"},
		Directory: workingDirectory,
		Environment: map[string]string{
			"RM_RELAY_RUNNER_HELPER": "1",
		},
	})
	if err != nil {
		t.Fatalf("Run() error = %v, stderr = %s", err, result.Stderr)
	}
	want := filepath.Clean(workingDirectory) + "\ncontrolled\n"
	if result.Stdout != want {
		t.Fatalf("Run() stdout = %q, want %q", result.Stdout, want)
	}
}

func TestOSRunnerHelperProcess(t *testing.T) {
	if os.Getenv("RM_RELAY_RUNNER_HELPER") != "1" {
		return
	}
	workingDirectory, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	fmt.Println(filepath.Clean(workingDirectory))
	if strings.Contains(os.Getenv("PATH"), string(os.PathListSeparator)) || os.Getenv("PATH") != "" {
		fmt.Println("controlled")
	}
	os.Exit(0)
}
