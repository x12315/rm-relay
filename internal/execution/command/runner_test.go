package command

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
	lines := strings.Split(strings.TrimSpace(result.Stdout), "\n")
	if len(lines) != 2 || lines[1] != "controlled" {
		t.Fatalf("Run() stdout = %q", result.Stdout)
	}
	actualDirectory, err := filepath.EvalSymlinks(lines[0])
	if err != nil {
		t.Fatal(err)
	}
	wantDirectory, err := filepath.EvalSymlinks(workingDirectory)
	if err != nil {
		t.Fatal(err)
	}
	if actualDirectory != wantDirectory {
		t.Fatalf("working directory = %q, want %q", actualDirectory, wantDirectory)
	}
}

func TestOSRunnerHonorsCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := (OSRunner{}).Run(ctx, Request{Name: os.Args[0], Arguments: []string{"-test.run=TestOSRunnerHelperProcess", "--"}})
	if err == nil {
		t.Fatal("Run() succeeded with a cancelled context")
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
