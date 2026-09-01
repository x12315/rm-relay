// Package buildkit executes local and remote workspace builds through named Buildx resources.
package buildkit

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/x12315/rm-relay/internal/build"
	"github.com/x12315/rm-relay/internal/builder"
	"github.com/x12315/rm-relay/internal/execution/buildx"
)

//go:embed workspace.Dockerfile
var workspaceDockerfile []byte

// Backend turns Project source into a locally exported Build Output through one Builder kind.
type Backend struct {
	kind      builder.Kind
	buildx    buildx.Client
	workflows build.WorkflowCatalog
	progress  io.Writer
}

// NewBackend binds the shared BuildKit solve to one supported Builder kind.
func NewBackend(kind builder.Kind, client buildx.Client, workflows build.WorkflowCatalog, progress io.Writer) (Backend, error) {
	if kind != builder.KindLocalBuildKit && kind != builder.KindRemoteBuildKit {
		return Backend{}, fmt.Errorf("BuildKit backend does not support builder kind %q", kind)
	}
	if client == nil {
		return Backend{}, fmt.Errorf("Buildx client is not configured")
	}
	return Backend{kind: kind, buildx: client, workflows: workflows, progress: progress}, nil
}

// Kind returns the execution mechanism implemented by this backend instance.
func (backend Backend) Kind() builder.Kind { return backend.kind }

// Build executes a solve and atomically publishes the returned install tree.
func (backend Backend) Build(ctx context.Context, plan build.Plan, definition builder.Definition) (build.ExecutionEvidence, error) {
	if definition.Kind != backend.kind {
		return build.ExecutionEvidence{}, fmt.Errorf("%s backend cannot execute builder kind %q", backend.kind, definition.Kind)
	}
	if definition.BuildxBuilder == "" {
		return build.ExecutionEvidence{}, fmt.Errorf("builder %q has no Buildx resource", definition.ID)
	}
	reference, err := definition.EnvironmentReference(plan.Profile.Config.Environment.ID)
	if err != nil {
		return build.ExecutionEvidence{}, err
	}
	if !builder.IsDigestReference(reference) {
		return build.ExecutionEvidence{}, fmt.Errorf("environment reference must be pinned by digest")
	}
	workflow, err := backend.workflows.Resolve(plan.Build.System)
	if err != nil {
		return build.ExecutionEvidence{}, err
	}
	task, err := workflow.Prepare(plan)
	if err != nil {
		return build.ExecutionEvidence{}, fmt.Errorf("prepare %s workspace: %w", plan.Build.System, err)
	}
	temporaryDirectory, publish, err := reserveOutputDirectory(plan.OutputDirectory)
	if err != nil {
		return build.ExecutionEvidence{}, err
	}
	defer publish(false)
	buildArguments := map[string]string{
		"RM_RELAY_ENVIRONMENT":  reference,
		"RM_RELAY_MISE_CONFIGS": strings.Join(task.MiseConfigFiles, ":"),
		"RM_RELAY_MISE_TASK":    task.Name,
		"RM_RELAY_BUILD_PRESET": plan.Build.Preset,
		"RM_RELAY_CCACHE_ID":    "rm-relay-ccache-" + plan.Profile.Digest,
	}
	request := buildx.BuildRequest{Builder: definition.BuildxBuilder, ContextDirectory: plan.ProjectRoot, OutputDirectory: temporaryDirectory, Dockerfile: bytes.NewReader(workspaceDockerfile), BuildArguments: buildArguments, Progress: backend.progress}
	if err := backend.buildx.Build(ctx, request); err != nil {
		return build.ExecutionEvidence{}, fmt.Errorf("workspace BuildKit solve: %w", err)
	}
	if err := validateDeclaredOutputs(temporaryDirectory, plan); err != nil {
		return build.ExecutionEvidence{}, err
	}
	if err := publish(true); err != nil {
		return build.ExecutionEvidence{}, err
	}
	digest := reference[strings.LastIndex(reference, "@")+1:]
	return build.ExecutionEvidence{BuilderID: definition.ID, BuilderKind: string(definition.Kind), EnvironmentID: plan.Profile.Config.Environment.ID, EnvironmentReference: reference, EnvironmentDigest: digest}, nil
}

func reserveOutputDirectory(outputDirectory string) (string, func(bool) error, error) {
	installRoot := filepath.Dir(outputDirectory)
	if err := os.MkdirAll(installRoot, 0o755); err != nil {
		return "", nil, fmt.Errorf("create install root: %w", err)
	}
	temporaryDirectory, err := os.MkdirTemp(installRoot, ".rm-relay-output-*")
	if err != nil {
		return "", nil, fmt.Errorf("create temporary Build Output: %w", err)
	}
	finalized := false
	publish := func(commit bool) error {
		if finalized {
			return nil
		}
		if !commit {
			return os.RemoveAll(temporaryDirectory)
		}
		if err := replaceDirectory(temporaryDirectory, outputDirectory); err != nil {
			return err
		}
		finalized = true
		return nil
	}
	return temporaryDirectory, publish, nil
}

func replaceDirectory(source, destination string) error {
	parent := filepath.Dir(destination)
	reservation, err := os.CreateTemp(parent, ".rm-relay-backup-*")
	if err != nil {
		return fmt.Errorf("reserve Build Output backup: %w", err)
	}
	backup := reservation.Name()
	if err := reservation.Close(); err != nil {
		return err
	}
	if err := os.Remove(backup); err != nil {
		return fmt.Errorf("prepare Build Output backup: %w", err)
	}
	hadPrevious := false
	if _, err := os.Stat(destination); err == nil {
		if err := os.Rename(destination, backup); err != nil {
			return fmt.Errorf("preserve previous Build Output: %w", err)
		}
		hadPrevious = true
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect previous Build Output: %w", err)
	}
	if err := os.Rename(source, destination); err != nil {
		if hadPrevious {
			if restoreError := os.Rename(backup, destination); restoreError != nil {
				return fmt.Errorf("publish Build Output: %w; restore previous output: %v", err, restoreError)
			}
		}
		return fmt.Errorf("publish Build Output: %w", err)
	}
	if hadPrevious {
		_ = os.RemoveAll(backup)
	}
	return nil
}

func validateDeclaredOutputs(root string, plan build.Plan) error {
	for _, output := range plan.Build.Outputs {
		if !filepath.IsLocal(output.Path) || filepath.Clean(output.Path) == "." {
			return fmt.Errorf("declared output %q is not a safe relative path", output.Path)
		}
		info, err := os.Lstat(filepath.Join(root, filepath.FromSlash(output.Path)))
		if err != nil {
			return fmt.Errorf("Build Output %q: %w", output.Role, err)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("Build Output %q must be a regular file", output.Role)
		}
	}
	return nil
}
