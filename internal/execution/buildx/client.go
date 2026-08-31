// Package buildx provides typed, shell-free Docker Buildx operations.
package buildx

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/x12315/rm-relay/internal/execution/command"
)

// CreateRemoteRequest registers an mTLS-protected BuildKit endpoint with Buildx.
type CreateRemoteRequest struct {
	Name, Endpoint, CAPath, CertificatePath, KeyPath, ServerName string
}

// BuildRequest describes one BuildKit solve and local export.
type BuildRequest struct {
	Builder, ContextDirectory, OutputDirectory string
	Dockerfile                                 io.Reader
	BuildArguments                             map[string]string
	Progress                                   io.Writer
}

// Client is the Buildx operation boundary consumed by Builder and backend modules.
type Client interface {
	CreateRemote(context.Context, CreateRemoteRequest) error
	RemoveBuilder(context.Context, string) error
	InspectBuilder(context.Context, string) error
	Build(context.Context, BuildRequest) error
}

// CLI executes Buildx operations through the shared process runner.
type CLI struct{ Runner command.Runner }

// CreateRemote registers one named remote driver. mTLS inputs are mandatory.
func (client CLI) CreateRemote(ctx context.Context, request CreateRemoteRequest) error {
	for name, value := range map[string]string{"name": request.Name, "endpoint": request.Endpoint, "CA": request.CAPath, "certificate": request.CertificatePath, "key": request.KeyPath, "server name": request.ServerName} {
		if value == "" {
			return fmt.Errorf("remote Builder %s must not be empty", name)
		}
	}
	if !strings.HasPrefix(request.Endpoint, "tcp://") {
		return fmt.Errorf("remote Builder endpoint must use tcp:// with mTLS")
	}
	for label, path := range map[string]string{"CA": request.CAPath, "certificate": request.CertificatePath, "key": request.KeyPath} {
		if !filepath.IsAbs(path) || strings.ContainsAny(path, ",\r\n") {
			return fmt.Errorf("remote Builder %s path must be absolute and contain no option separators", label)
		}
		info, err := os.Stat(path)
		if err != nil || !info.Mode().IsRegular() {
			return fmt.Errorf("remote Builder %s path %q is not a readable regular file", label, path)
		}
	}
	for label, value := range map[string]string{"name": request.Name, "endpoint": request.Endpoint, "server name": request.ServerName} {
		if strings.ContainsAny(value, ",\r\n") {
			return fmt.Errorf("remote Builder %s contains an invalid separator", label)
		}
	}
	driverOptions := strings.Join([]string{"cacert=" + request.CAPath, "cert=" + request.CertificatePath, "key=" + request.KeyPath, "servername=" + request.ServerName}, ",")
	result, err := client.run(ctx, command.Request{Name: "docker", Arguments: []string{"buildx", "create", "--name", request.Name, "--driver", "remote", "--driver-opt", driverOptions, request.Endpoint}})
	if err != nil {
		return failure("create remote Buildx builder", result, err)
	}
	return nil
}

// RemoveBuilder removes one explicit Buildx builder registration.
func (client CLI) RemoveBuilder(ctx context.Context, name string) error {
	result, err := client.run(ctx, command.Request{Name: "docker", Arguments: []string{"buildx", "rm", name}})
	if err != nil {
		return failure("remove Buildx builder", result, err)
	}
	return nil
}

// InspectBuilder asks Buildx to bootstrap and validate its driver.
func (client CLI) InspectBuilder(ctx context.Context, name string) error {
	result, err := client.run(ctx, command.Request{Name: "docker", Arguments: []string{"buildx", "inspect", "--bootstrap", name}})
	if err != nil {
		return failure("inspect Buildx builder", result, err)
	}
	return nil
}

// Build runs one solve and exports files directly to a caller-owned local directory.
func (client CLI) Build(ctx context.Context, request BuildRequest) error {
	if request.Builder == "" || request.ContextDirectory == "" || request.OutputDirectory == "" || request.Dockerfile == nil {
		return fmt.Errorf("Buildx build requires builder, context, Dockerfile, and output directory")
	}
	arguments := []string{"buildx", "build", "--builder", request.Builder, "--file", "-", "--progress", "plain", "--output", "type=local,dest=" + request.OutputDirectory}
	keys := sortedKeys(request.BuildArguments)
	for _, key := range keys {
		arguments = append(arguments, "--build-arg", key+"="+request.BuildArguments[key])
	}
	arguments = append(arguments, request.ContextDirectory)
	result, err := client.run(ctx, command.Request{Name: "docker", Arguments: arguments, Stdin: request.Dockerfile, Stdout: request.Progress, Stderr: request.Progress})
	if err != nil {
		return failure("run Buildx build", result, err)
	}
	return nil
}

func (client CLI) run(ctx context.Context, request command.Request) (command.Result, error) {
	if client.Runner == nil {
		return command.Result{}, fmt.Errorf("process runner is not configured")
	}
	return client.Runner.Run(ctx, request)
}

func sortedKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j] < keys[j-1]; j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
	return keys
}

func failure(action string, result command.Result, err error) error {
	details := strings.TrimSpace(result.Stderr)
	if details == "" {
		return fmt.Errorf("%s: %w", action, err)
	}
	return fmt.Errorf("%s: %w: %s", action, err, details)
}
