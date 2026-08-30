// Package command is RM Relay's only boundary for launching operating-system processes.
package command

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"
)

// Request describes one process without passing through a command shell.
type Request struct {
	Name        string
	Arguments   []string
	Directory   string
	Environment map[string]string
	Stdout      io.Writer
	Stderr      io.Writer
}

// Result contains captured process output even when the process fails.
type Result struct {
	Stdout string
	Stderr string
}

// Runner executes process requests. Tests replace it to inspect native argument boundaries.
type Runner interface {
	Run(context.Context, Request) (Result, error)
}

// OSRunner launches a process through os/exec and captures its output.
type OSRunner struct{}

// Run executes request with the current environment plus explicit key overrides.
func (OSRunner) Run(ctx context.Context, request Request) (Result, error) {
	if request.Name == "" {
		return Result{}, fmt.Errorf("process name must not be empty")
	}
	command := exec.CommandContext(ctx, request.Name, request.Arguments...)
	command.Dir = request.Directory
	command.Env = mergedEnvironment(request.Environment)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = captureAndForward(&stdout, request.Stdout)
	command.Stderr = captureAndForward(&stderr, request.Stderr)
	err := command.Run()
	result := Result{Stdout: stdout.String(), Stderr: stderr.String()}
	if err != nil {
		return result, fmt.Errorf("run %s: %w", request.Name, err)
	}
	return result, nil
}

func captureAndForward(capture *bytes.Buffer, destination io.Writer) io.Writer {
	if destination == nil {
		return capture
	}
	return io.MultiWriter(capture, destination)
}

func mergedEnvironment(overrides map[string]string) []string {
	values := make(map[string]string, len(os.Environ())+len(overrides))
	for _, assignment := range os.Environ() {
		key, value, found := strings.Cut(assignment, "=")
		if found {
			values[key] = value
		}
	}
	for key, value := range overrides {
		values[key] = value
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	environment := make([]string, 0, len(keys))
	for _, key := range keys {
		environment = append(environment, key+"="+values[key])
	}
	return environment
}
