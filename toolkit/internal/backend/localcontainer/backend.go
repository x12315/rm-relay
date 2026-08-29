// Package localcontainer executes project mise tasks in a fixed local Docker image.
package localcontainer

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"

	"github.com/x12315/rm-relay/toolkit/internal/commandexec"
	"github.com/x12315/rm-relay/toolkit/internal/executionplan"
	"github.com/x12315/rm-relay/toolkit/internal/miseexec"
)

const (
	containerProjectRoot = "/workspace"
	containerAssetsRoot  = "/opt/rm-relay"
)

// Backend maps an Execution Plan to the local Docker CLI.
type Backend struct {
	Runner   commandexec.Runner
	Progress io.Writer
}

// Build runs the declared workspace task and returns the inspected development image identity.
func (backend Backend) Build(ctx context.Context, plan executionplan.Plan) (string, error) {
	if backend.Runner == nil {
		return "", fmt.Errorf("process runner is not configured")
	}
	if plan.Backend != "local" {
		return "", fmt.Errorf("local container backend cannot execute backend %q", plan.Backend)
	}
	imageReference := plan.Profile.Config.DevelopmentImage
	inspectResult, err := backend.Runner.Run(ctx, commandexec.Request{
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

	taskInvocation := miseexec.TaskInvocation(
		[]string{
			containerAssetsRoot + "/mise/core.toml",
			containerAssetsRoot + "/profiles/" + plan.Profile.Config.ID + "/" + filepath.ToSlash(plan.Profile.Config.MiseConfig),
			containerProjectRoot + "/" + filepath.ToSlash(plan.Build.MiseConfig),
		},
		plan.Build.Task,
		":",
	)
	taskInvocation.Environment["RM_RELAY_OUTPUT_DIR"] = containerProjectRoot + "/install/" + plan.Profile.Config.ID
	arguments := []string{
		"run",
		"--rm",
		"--volume", plan.ProjectRoot + ":" + containerProjectRoot,
		"--volume", plan.AssetsRoot + ":" + containerAssetsRoot + ":ro",
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
	buildResult, err := backend.Runner.Run(ctx, commandexec.Request{
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

func processFailure(action string, result commandexec.Result, processError error) error {
	details := strings.TrimSpace(result.Stderr)
	if details == "" {
		return fmt.Errorf("%s: %w", action, processError)
	}
	return fmt.Errorf("%s: %w: %s", action, processError, details)
}
