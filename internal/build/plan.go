// Package build resolves project declarations into backend-independent build plans.
package build

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/x12315/rm-relay/internal/profile"
	"github.com/x12315/rm-relay/internal/project"
)

var (
	// ErrProject marks failures in the user-owned project declaration.
	ErrProject = errors.New("project resolution failed")
	// ErrProfile marks failures in the RM Relay-owned Profile catalog.
	ErrProfile = errors.New("profile resolution failed")
)

// Operation identifies the user action represented by a Plan.
type Operation string

const (
	// OperationBuild produces a Build Output from the project workspace.
	OperationBuild Operation = "build"
	// OperationFlash consumes an existing Build Output for an MCU target.
	OperationFlash Operation = "flash"
)

// Plan is the backend-independent handoff between project, build and target modules.
type Plan struct {
	Operation       Operation
	ProjectRoot     string
	ProjectID       string
	BuilderID       string
	Profile         profile.Loaded
	Build           project.Build
	OutputDirectory string
}

// Resolve loads one project/Profile combination and returns deterministic absolute paths.
func Resolve(operation Operation, projectRoot, profileOverride, builderOverride string, profiles profile.Catalog) (Plan, error) {
	if operation != OperationBuild && operation != OperationFlash {
		return Plan{}, fmt.Errorf("unsupported operation %q", operation)
	}
	projectRoot, err := filepath.Abs(projectRoot)
	if err != nil {
		return Plan{}, fmt.Errorf("resolve project root: %w", err)
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
	builderID := builderOverride
	if builderID == "" {
		builderID = projectConfig.DefaultBuilder
	}
	loadedProfile, err := profiles.Load(profileID)
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
	return Plan{
		Operation:       operation,
		ProjectRoot:     projectRoot,
		ProjectID:       projectConfig.ProjectID,
		BuilderID:       builderID,
		Profile:         loadedProfile,
		Build:           buildDeclaration,
		OutputDirectory: filepath.Join(projectRoot, "install", profileID),
	}, nil
}

func requireProfileOutputs(declaration project.Build, requiredRoles []string) error {
	declaredRoles := make(map[string]struct{}, len(declaration.Outputs))
	for _, declaredOutput := range declaration.Outputs {
		declaredRoles[declaredOutput.Role] = struct{}{}
	}
	for _, role := range requiredRoles {
		if _, exists := declaredRoles[role]; !exists {
			return fmt.Errorf("project build %q does not declare required output role %q", declaration.Profile, role)
		}
	}
	return nil
}
