// Package localcontainer executes workspace builds in fixed local Docker images.
package localcontainer

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/x12315/rm-relay/internal/build"
	"github.com/x12315/rm-relay/internal/builder"
	"github.com/x12315/rm-relay/internal/execution/docker"
	"github.com/x12315/rm-relay/internal/execution/mise"
)

const (
	// ID is retained as the stable backend kind for cache path compatibility.
	ID                   = string(builder.KindLocalContainer)
	containerProjectRoot = "/workspace"
	containerCacheRoot   = "/cache"
)

// Backend maps a build Plan to the local Docker CLI.
type Backend struct {
	Docker         docker.Client
	Workflows      build.WorkflowCatalog
	CacheDirectory string
	Progress       io.Writer
}

// Kind returns the execution mechanism implemented by this backend.
func (Backend) Kind() builder.Kind {
	return builder.KindLocalContainer
}

// Build runs the selected workspace workflow and returns execution evidence.
func (backend Backend) Build(ctx context.Context, plan build.Plan, definition builder.Definition) (build.ExecutionEvidence, error) {
	if backend.Docker == nil {
		return build.ExecutionEvidence{}, fmt.Errorf("Docker client is not configured")
	}
	if definition.Kind != builder.KindLocalContainer {
		return build.ExecutionEvidence{}, fmt.Errorf("local backend cannot execute builder kind %q", definition.Kind)
	}
	workflow, err := backend.Workflows.Resolve(plan.Build.System)
	if err != nil {
		return build.ExecutionEvidence{}, err
	}
	workspaceTask, err := workflow.Prepare(plan)
	if err != nil {
		return build.ExecutionEvidence{}, fmt.Errorf("prepare %s workspace: %w", plan.Build.System, err)
	}
	if backend.CacheDirectory == "" {
		return build.ExecutionEvidence{}, fmt.Errorf("local build cache directory is not configured")
	}
	if err := os.MkdirAll(backend.CacheDirectory, 0o755); err != nil {
		return build.ExecutionEvidence{}, fmt.Errorf("create local build cache: %w", err)
	}

	imageReference, err := definition.EnvironmentReference(plan.Profile.Config.Environment.ID, plan.Profile.Config.Environment.LocalReference)
	if err != nil {
		return build.ExecutionEvidence{}, err
	}
	imageID, err := backend.Docker.InspectImage(ctx, imageReference)
	if err != nil {
		return build.ExecutionEvidence{}, fmt.Errorf("inspect development environment: %w", err)
	}

	taskInvocation := mise.TaskInvocation(workspaceTask.MiseConfigFiles, workspaceTask.Name, ":")
	for key, value := range workspaceTask.Environment {
		taskInvocation.Environment[key] = value
	}
	taskInvocation.Environment["CCACHE_DIR"] = containerCacheRoot + "/ccache"
	if err := backend.Docker.Run(ctx, docker.RunRequest{
		Image:       imageReference,
		Volumes:     []string{plan.ProjectRoot + ":" + containerProjectRoot, backend.CacheDirectory + ":" + containerCacheRoot},
		Workdir:     containerProjectRoot,
		Environment: taskInvocation.Environment,
		Command:     append([]string{"mise"}, taskInvocation.Arguments...),
		Stdout:      backend.Progress,
		Stderr:      backend.Progress,
	}); err != nil {
		return build.ExecutionEvidence{}, fmt.Errorf("run workspace build: %w", err)
	}
	return build.ExecutionEvidence{
		BuilderID: definition.ID, BuilderKind: string(definition.Kind),
		EnvironmentID: plan.Profile.Config.Environment.ID, EnvironmentReference: imageReference,
		EnvironmentDigest: imageID,
	}, nil
}
