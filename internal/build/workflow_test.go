package build

import (
	"strings"
	"testing"
)

func TestWorkflowCatalogResolvesRegisteredSystem(t *testing.T) {
	catalog, err := NewWorkflowCatalog(fakeWorkflow{system: "cmake"})
	if err != nil {
		t.Fatal(err)
	}

	workflow, err := catalog.Resolve("cmake")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if workflow.System() != "cmake" {
		t.Fatalf("System() = %q", workflow.System())
	}
}

func TestWorkflowCatalogRejectsUnknownSystemWithAvailableChoices(t *testing.T) {
	catalog, err := NewWorkflowCatalog(fakeWorkflow{system: "cmake"})
	if err != nil {
		t.Fatal(err)
	}

	_, err = catalog.Resolve("colcon")
	if err == nil || !strings.Contains(err.Error(), "cmake") {
		t.Fatalf("Resolve() error = %v, want available system", err)
	}
}

type fakeWorkflow struct {
	system string
}

func (workflow fakeWorkflow) System() string {
	return workflow.system
}

func (fakeWorkflow) Prepare(Plan) (WorkspaceTask, error) {
	return WorkspaceTask{}, nil
}
