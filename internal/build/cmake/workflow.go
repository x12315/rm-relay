// Package cmake translates RM Relay CMake build declarations into controlled workspace tasks.
package cmake

import (
	"fmt"

	"github.com/x12315/rm-relay/internal/build"
	"github.com/x12315/rm-relay/internal/execution/mise"
)

const (
	System              = "cmake"
	containerConfigPath = "/opt/rm-relay/build/cmake/build.mise.toml"
	workspaceTaskName   = "rm-relay:build"
)

// Workflow implements the CMake Configure, Build and Install path used by RM Relay images.
type Workflow struct{}

func (Workflow) System() string {
	return System
}

// Prepare maps a project preset to the fixed mise task installed in a development image.
func (Workflow) Prepare(plan build.Plan) (build.WorkspaceTask, error) {
	if plan.Build.System != System {
		return build.WorkspaceTask{}, fmt.Errorf("CMake workflow cannot prepare build system %q", plan.Build.System)
	}
	return build.WorkspaceTask{
		Name:            workspaceTaskName,
		MiseConfigFiles: []string{mise.ContainerBaseConfig, containerConfigPath},
		Environment: map[string]string{
			"RM_RELAY_BUILD_PRESET": plan.Build.Preset,
			"RM_RELAY_OUTPUT_DIR":   "/workspace/install/" + plan.Profile.Config.ID,
		},
	}, nil
}
