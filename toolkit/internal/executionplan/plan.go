// Package executionplan resolves validated project and Profile declarations into backend inputs.
package executionplan

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/x12315/rm-relay/toolkit/internal/profile"
	"github.com/x12315/rm-relay/toolkit/internal/project"
)

var (
	// ErrProject marks failures in the user-owned project declaration.
	ErrProject = errors.New("project resolution failed")
	// ErrProfile marks failures in the RM Relay-owned Profile catalog or assets.
	ErrProfile = errors.New("profile resolution failed")
)

// Operation identifies the user action represented by a Plan.
type Operation string

const (
	// OperationBuild produces a Build Output from the project workspace.
	OperationBuild Operation = "build"
	// OperationFlash sends an existing Build Output to an MCU target.
	OperationFlash Operation = "flash"
)

// Plan is the fully resolved, internal handoff between CLI, backend and target modules.
type Plan struct {
	Operation         Operation
	ProjectRoot       string
	ProjectID         string
	AssetsRoot        string
	Profile           profile.Loaded
	Build             project.Build
	Backend           string
	OutputDirectory   string
	CoreMiseConfig    string
	ProfileMiseConfig string
	ProjectMiseConfig string
}

// Resolve loads one project/Profile combination and returns deterministic absolute paths.
func Resolve(operation Operation, projectRoot, assetsRoot, profileOverride string) (Plan, error) {
	if operation != OperationBuild && operation != OperationFlash {
		return Plan{}, fmt.Errorf("unsupported operation %q", operation)
	}
	projectRoot, err := filepath.Abs(projectRoot)
	if err != nil {
		return Plan{}, fmt.Errorf("resolve project root: %w", err)
	}
	assetsRoot, err = filepath.Abs(assetsRoot)
	if err != nil {
		return Plan{}, fmt.Errorf("resolve RM Relay assets: %w", err)
	}
	projectConfig, err := project.Load(projectRoot)
	if err != nil {
		return Plan{}, fmt.Errorf("%w: %v", ErrProject, err)
	}
	if projectConfig.ProjectID == "" {
		return Plan{}, fmt.Errorf("%w: project identity is empty; run rm-relay init first", ErrProject)
	}
	profileID := profileOverride
	if profileID == "" {
		profileID = projectConfig.DefaultProfile
	}
	loadedProfile, err := (profile.Catalog{
		ProfilesRoot: filepath.Join(assetsRoot, "profiles"),
		AssetsRoot:   assetsRoot,
	}).Load(profileID)
	if err != nil {
		return Plan{}, fmt.Errorf("%w: %v", ErrProfile, err)
	}
	buildDeclaration, err := projectConfig.BuildForProfile(profileID)
	if err != nil {
		return Plan{}, fmt.Errorf("%w: %v", ErrProject, err)
	}
	if err := requireProfileOutputs(buildDeclaration, loadedProfile.Config.RequiredOutputRoles); err != nil {
		return Plan{}, fmt.Errorf("%w: %v", ErrProject, err)
	}

	coreMiseConfig := filepath.Join(assetsRoot, "mise", "core.toml")
	profileMiseConfig := filepath.Join(loadedProfile.Directory, loadedProfile.Config.MiseConfig)
	projectMiseConfig := filepath.Join(projectRoot, buildDeclaration.MiseConfig)
	if err := requireRegularFile(coreMiseConfig); err != nil {
		return Plan{}, fmt.Errorf("%w: core mise config: %v", ErrProfile, err)
	}
	if err := requireRegularFile(profileMiseConfig); err != nil {
		return Plan{}, fmt.Errorf("%w: profile mise config: %v", ErrProfile, err)
	}
	if err := requireRegularFile(projectMiseConfig); err != nil {
		return Plan{}, fmt.Errorf("%w: project mise config: %v", ErrProject, err)
	}
	return Plan{
		Operation:         operation,
		ProjectRoot:       projectRoot,
		ProjectID:         projectConfig.ProjectID,
		AssetsRoot:        assetsRoot,
		Profile:           loadedProfile,
		Build:             buildDeclaration,
		Backend:           "local",
		OutputDirectory:   filepath.Join(projectRoot, "install", profileID),
		CoreMiseConfig:    coreMiseConfig,
		ProfileMiseConfig: profileMiseConfig,
		ProjectMiseConfig: projectMiseConfig,
	}, nil
}

// MiseConfigs returns the controlled config stack in least-specific to most-specific order.
func (plan Plan) MiseConfigs() []string {
	return []string{plan.CoreMiseConfig, plan.ProfileMiseConfig, plan.ProjectMiseConfig}
}

func requireProfileOutputs(build project.Build, requiredRoles []string) error {
	declaredRoles := make(map[string]struct{}, len(build.Outputs))
	for _, output := range build.Outputs {
		declaredRoles[output.Role] = struct{}{}
	}
	for _, role := range requiredRoles {
		if _, exists := declaredRoles[role]; !exists {
			return fmt.Errorf("project build %q does not declare required output role %q", build.Profile, role)
		}
	}
	return nil
}

func requireRegularFile(path string) error {
	fileInfo, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("stat %s: %w", path, err)
	}
	if fileInfo.Mode()&os.ModeSymlink != 0 || !fileInfo.Mode().IsRegular() {
		return fmt.Errorf("%s must be a regular file", path)
	}
	return nil
}
