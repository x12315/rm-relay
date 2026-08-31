// Package remotebuildkit builds workspaces through named Buildx remote builders.
package remotebuildkit

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

// Backend turns Project source into a locally exported Build Output through BuildKit.
type Backend struct {
	Buildx    buildx.Client
	Workflows build.WorkflowCatalog
	Progress  io.Writer
}

// Kind returns the execution mechanism implemented by this backend.
func (Backend) Kind() builder.Kind { return builder.KindRemoteBuildKit }

// Build executes a remote solve and atomically publishes the returned install tree.
func (backend Backend) Build(ctx context.Context, plan build.Plan, definition builder.Definition) (build.ExecutionEvidence, error) {
	if backend.Buildx == nil {
		return build.ExecutionEvidence{}, fmt.Errorf("Buildx client is not configured")
	}
	if definition.Kind != builder.KindRemoteBuildKit {
		return build.ExecutionEvidence{}, fmt.Errorf("remote backend cannot execute builder kind %q", definition.Kind)
	}
	if definition.BuildxBuilder == "" {
		return build.ExecutionEvidence{}, fmt.Errorf("builder %q has no Buildx resource", definition.ID)
	}
	reference, err := definition.EnvironmentReference(plan.Profile.Config.Environment.ID, plan.Profile.Config.Environment.LocalReference)
	if err != nil {
		return build.ExecutionEvidence{}, err
	}
	if !builder.IsDigestReference(reference) {
		return build.ExecutionEvidence{}, fmt.Errorf("remote environment reference must be pinned by digest")
	}
	workflow, err := backend.Workflows.Resolve(plan.Build.System)
	if err != nil {
		return build.ExecutionEvidence{}, err
	}
	task, err := workflow.Prepare(plan)
	if err != nil {
		return build.ExecutionEvidence{}, fmt.Errorf("prepare %s workspace: %w", plan.Build.System, err)
	}
	installRoot := filepath.Dir(plan.OutputDirectory)
	if err := os.MkdirAll(installRoot, 0o755); err != nil {
		return build.ExecutionEvidence{}, fmt.Errorf("create install root: %w", err)
	}
	temporaryDirectory, err := os.MkdirTemp(installRoot, ".rm-relay-output-*")
	if err != nil {
		return build.ExecutionEvidence{}, fmt.Errorf("create temporary Build Output: %w", err)
	}
	published := false
	defer func() {
		if !published {
			_ = os.RemoveAll(temporaryDirectory)
		}
	}()
	buildArguments := map[string]string{
		"RM_RELAY_ENVIRONMENT":  reference,
		"RM_RELAY_MISE_CONFIGS": strings.Join(task.MiseConfigFiles, ":"),
		"RM_RELAY_MISE_TASK":    task.Name,
		"RM_RELAY_BUILD_PRESET": plan.Build.Preset,
		"RM_RELAY_CCACHE_ID":    "rm-relay-ccache-" + plan.Profile.Digest,
	}
	if err := backend.Buildx.Build(ctx, buildx.BuildRequest{Builder: definition.BuildxBuilder, ContextDirectory: plan.ProjectRoot, OutputDirectory: temporaryDirectory, Dockerfile: bytes.NewReader(workspaceDockerfile), BuildArguments: buildArguments, Progress: backend.Progress}); err != nil {
		return build.ExecutionEvidence{}, fmt.Errorf("remote workspace build: %w", err)
	}
	if err := validateDeclaredOutputs(temporaryDirectory, plan); err != nil {
		return build.ExecutionEvidence{}, err
	}
	backupReservation, err := os.CreateTemp(installRoot, ".rm-relay-backup-*")
	if err != nil {
		return build.ExecutionEvidence{}, fmt.Errorf("reserve Build Output backup: %w", err)
	}
	backupDirectory := backupReservation.Name()
	if err := backupReservation.Close(); err != nil {
		return build.ExecutionEvidence{}, err
	}
	if err := os.Remove(backupDirectory); err != nil {
		return build.ExecutionEvidence{}, fmt.Errorf("prepare Build Output backup: %w", err)
	}
	hadPrevious := false
	if _, err := os.Stat(plan.OutputDirectory); err == nil {
		if err := os.Rename(plan.OutputDirectory, backupDirectory); err != nil {
			return build.ExecutionEvidence{}, fmt.Errorf("preserve previous Build Output: %w", err)
		}
		hadPrevious = true
	} else if !os.IsNotExist(err) {
		return build.ExecutionEvidence{}, fmt.Errorf("inspect previous Build Output: %w", err)
	}
	if err := os.Rename(temporaryDirectory, plan.OutputDirectory); err != nil {
		if hadPrevious {
			if restoreError := os.Rename(backupDirectory, plan.OutputDirectory); restoreError != nil {
				return build.ExecutionEvidence{}, fmt.Errorf("publish remote Build Output: %w; restore previous output: %v", err, restoreError)
			}
		}
		return build.ExecutionEvidence{}, fmt.Errorf("publish remote Build Output: %w", err)
	}
	published = true
	if hadPrevious {
		_ = os.RemoveAll(backupDirectory)
	}
	digest := reference[strings.LastIndex(reference, "@")+1:]
	return build.ExecutionEvidence{BuilderID: definition.ID, BuilderKind: string(definition.Kind), EnvironmentID: plan.Profile.Config.Environment.ID, EnvironmentReference: reference, EnvironmentDigest: digest}, nil
}

func validateDeclaredOutputs(root string, plan build.Plan) error {
	for _, output := range plan.Build.Outputs {
		if !filepath.IsLocal(output.Path) || filepath.Clean(output.Path) == "." {
			return fmt.Errorf("declared output %q is not a safe relative path", output.Path)
		}
		info, err := os.Lstat(filepath.Join(root, filepath.FromSlash(output.Path)))
		if err != nil {
			return fmt.Errorf("remote output %q: %w", output.Role, err)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("remote output %q must be a regular file", output.Role)
		}
	}
	return nil
}
