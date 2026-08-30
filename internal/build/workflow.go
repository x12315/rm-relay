package build

import (
	"fmt"
	"sort"
)

// WorkspaceTask is the controlled mise task a backend executes for one build plan.
type WorkspaceTask struct {
	Name            string
	MiseConfigFiles []string
	Environment     map[string]string
}

// Workflow translates one build-system declaration into a controlled workspace task.
type Workflow interface {
	System() string
	Prepare(Plan) (WorkspaceTask, error)
}

// WorkflowCatalog selects a build-system implementation by the stable system ID in rm-relay.toml.
type WorkflowCatalog struct {
	bySystem map[string]Workflow
}

// NewWorkflowCatalog validates and indexes the supplied build-system workflows.
func NewWorkflowCatalog(workflows ...Workflow) (WorkflowCatalog, error) {
	bySystem := make(map[string]Workflow, len(workflows))
	for _, workflow := range workflows {
		if workflow == nil {
			return WorkflowCatalog{}, fmt.Errorf("build workflow must not be nil")
		}
		system := workflow.System()
		if system == "" {
			return WorkflowCatalog{}, fmt.Errorf("build workflow system must not be empty")
		}
		if _, exists := bySystem[system]; exists {
			return WorkflowCatalog{}, fmt.Errorf("multiple build workflows use system %q", system)
		}
		bySystem[system] = workflow
	}
	return WorkflowCatalog{bySystem: bySystem}, nil
}

// Resolve returns the workflow registered for system.
func (catalog WorkflowCatalog) Resolve(system string) (Workflow, error) {
	workflow, exists := catalog.bySystem[system]
	if !exists {
		available := make([]string, 0, len(catalog.bySystem))
		for registeredSystem := range catalog.bySystem {
			available = append(available, registeredSystem)
		}
		sort.Strings(available)
		return nil, fmt.Errorf("unsupported build system %q; available systems: %v", system, available)
	}
	return workflow, nil
}
