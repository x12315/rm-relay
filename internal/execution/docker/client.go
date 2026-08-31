// Package docker provides typed, shell-free Docker CLI operations.
package docker

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/x12315/rm-relay/internal/execution/command"
)

// RunRequest describes an ephemeral container without exposing raw Docker arguments.
type RunRequest struct {
	Image       string
	Volumes     []string
	Workdir     string
	Environment map[string]string
	Command     []string
	Stdout      io.Writer
	Stderr      io.Writer
}

// Client is the Docker Engine operation boundary consumed by product modules.
type Client interface {
	CheckEngine(context.Context) error
	InspectImage(context.Context, string) (string, error)
	Run(context.Context, RunRequest) error
	TagImage(context.Context, string, string) error
	RemoveImage(context.Context, string) error
}

// CLI executes Docker operations through the shared process runner.
type CLI struct{ Runner command.Runner }

// CheckEngine verifies that the Docker client can reach a running Engine.
func (client CLI) CheckEngine(ctx context.Context) error {
	result, err := client.run(ctx, command.Request{Name: "docker", Arguments: []string{"version", "--format", "{{.Server.Version}}"}})
	if err != nil {
		return failure("check Docker Engine", result, err)
	}
	if strings.TrimSpace(result.Stdout) == "" {
		return fmt.Errorf("Docker Engine returned an empty version")
	}
	return nil
}

// InspectImage returns one immutable Docker image ID.
func (client CLI) InspectImage(ctx context.Context, reference string) (string, error) {
	result, err := client.run(ctx, command.Request{Name: "docker", Arguments: []string{"image", "inspect", "--format", "{{.Id}}", reference}})
	if err != nil {
		return "", failure("inspect image", result, err)
	}
	fields := strings.Fields(result.Stdout)
	if len(fields) != 1 {
		return "", fmt.Errorf("inspect image returned %d identities", len(fields))
	}
	return fields[0], nil
}

// Run starts and removes one container after completion.
func (client CLI) Run(ctx context.Context, request RunRequest) error {
	arguments := []string{"run", "--rm"}
	for _, volume := range request.Volumes {
		arguments = append(arguments, "--volume", volume)
	}
	if request.Workdir != "" {
		arguments = append(arguments, "--workdir", request.Workdir)
	}
	keys := make([]string, 0, len(request.Environment))
	for key := range request.Environment {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		arguments = append(arguments, "--env", key+"="+request.Environment[key])
	}
	arguments = append(arguments, request.Image)
	arguments = append(arguments, request.Command...)
	result, err := client.run(ctx, command.Request{Name: "docker", Arguments: arguments, Stdout: request.Stdout, Stderr: request.Stderr})
	if err != nil {
		return failure("run container", result, err)
	}
	return nil
}

// TagImage creates a local image alias.
func (client CLI) TagImage(ctx context.Context, source, target string) error {
	result, err := client.run(ctx, command.Request{Name: "docker", Arguments: []string{"image", "tag", source, target}})
	if err != nil {
		return failure("tag image", result, err)
	}
	return nil
}

// RemoveImage removes one explicit local image reference.
func (client CLI) RemoveImage(ctx context.Context, reference string) error {
	result, err := client.run(ctx, command.Request{Name: "docker", Arguments: []string{"image", "rm", reference}})
	if err != nil {
		return failure("remove image", result, err)
	}
	return nil
}

func (client CLI) run(ctx context.Context, request command.Request) (command.Result, error) {
	if client.Runner == nil {
		return command.Result{}, fmt.Errorf("process runner is not configured")
	}
	return client.Runner.Run(ctx, request)
}

func failure(action string, result command.Result, err error) error {
	details := strings.TrimSpace(result.Stderr)
	if details == "" {
		return fmt.Errorf("%s: %w", action, err)
	}
	return fmt.Errorf("%s: %w: %s", action, err, details)
}
