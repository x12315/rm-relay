// Package localcontainer executes workspace builds in fixed local Docker images.
package localcontainer

import (
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/x12315/rm-relay/internal/build"
	"github.com/x12315/rm-relay/internal/execution/command"
	"github.com/x12315/rm-relay/internal/execution/mise"
)

const (
	// ID is the stable backend identifier used by CLI and service composition.
	ID                   = "local-container"
	containerProjectRoot = "/workspace"
	containerCacheRoot   = "/cache"
)

// Backend maps a build Plan to the local Docker CLI.
type Backend struct {
	Runner         command.Runner
	Workflows      build.WorkflowCatalog
	CacheDirectory string
	Progress       io.Writer
}

// ID returns the stable backend identifier used in diagnostics and selection layers.
func (Backend) ID() string {
	return ID
}

// Build runs the selected workspace workflow and returns the development image identity.
func (backend Backend) Build(ctx context.Context, plan build.Plan) (string, error) {
	if backend.Runner == nil {
		return "", fmt.Errorf("process runner is not configured")
	}
	workflow, err := backend.Workflows.Resolve(plan.Build.System)
	if err != nil {
		return "", err
	}
	workspaceTask, err := workflow.Prepare(plan)
	if err != nil {
		return "", fmt.Errorf("prepare %s workspace: %w", plan.Build.System, err)
	}
	if backend.CacheDirectory == "" {
		return "", fmt.Errorf("local build cache directory is not configured")
	}
	if err := os.MkdirAll(backend.CacheDirectory, 0o755); err != nil {
		return "", fmt.Errorf("create local build cache: %w", err)
	}

	imageReference := plan.Profile.Config.DevelopmentImage
	inspectResult, err := backend.Runner.Run(ctx, command.Request{
		Name:      "docker",
		Arguments: []string{"image", "inspect", "--format", "{{.Id}}", imageReference},
		Stderr:    backend.Progress,
	})
	if err != nil {
		return "", processFailure("inspect development image", inspectResult, err)
	}
	imageIDFields := strings.Fields(inspectResult.Stdout)
	if len(imageIDFields) != 1 {
		return "", fmt.Errorf("inspect development image returned %d identities", len(imageIDFields))
	}

	taskInvocation := mise.TaskInvocation(workspaceTask.MiseConfigFiles, workspaceTask.Name, ":")
	for key, value := range workspaceTask.Environment {
		taskInvocation.Environment[key] = value
	}
	taskInvocation.Environment["CCACHE_DIR"] = containerCacheRoot + "/ccache"
	arguments := []string{
		"run",
		"--rm",
		"--volume", plan.ProjectRoot + ":" + containerProjectRoot,
		"--volume", backend.CacheDirectory + ":" + containerCacheRoot,
		"--workdir", containerProjectRoot,
	}
	environmentKeys := make([]string, 0, len(taskInvocation.Environment))
	for key := range taskInvocation.Environment {
		environmentKeys = append(environmentKeys, key)
	}
	sort.Strings(environmentKeys)
	for _, key := range environmentKeys {
		arguments = append(arguments, "--env", key+"="+taskInvocation.Environment[key])
	}
	arguments = append(arguments, imageReference, "mise")
	arguments = append(arguments, taskInvocation.Arguments...)
	buildResult, err := backend.Runner.Run(ctx, command.Request{
		Name:      "docker",
		Arguments: arguments,
		Stdout:    backend.Progress,
		Stderr:    backend.Progress,
	})
	if err != nil {
		return "", processFailure("run workspace build", buildResult, err)
	}
	return imageIDFields[0], nil
}

func processFailure(action string, result command.Result, processError error) error {
	details := strings.TrimSpace(result.Stderr)
	if details == "" {
		return fmt.Errorf("%s: %w", action, processError)
	}
	return fmt.Errorf("%s: %w: %s", action, processError, details)
}
