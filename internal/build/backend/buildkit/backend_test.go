package buildkit

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/x12315/rm-relay/internal/build"
	"github.com/x12315/rm-relay/internal/build/cmake"
	"github.com/x12315/rm-relay/internal/builder"
	"github.com/x12315/rm-relay/internal/execution/buildx"
	"github.com/x12315/rm-relay/internal/profile"
	"github.com/x12315/rm-relay/internal/project"
)

type fakeBuildx struct{ request buildx.BuildRequest }

func (*fakeBuildx) ListBuilders(context.Context) ([]buildx.BuilderSummary, error)  { return nil, nil }
func (*fakeBuildx) CreateLocal(context.Context, buildx.CreateLocalRequest) error   { return nil }
func (*fakeBuildx) CreateRemote(context.Context, buildx.CreateRemoteRequest) error { return nil }
func (*fakeBuildx) RemoveBuilder(context.Context, string) error                    { return nil }
func (*fakeBuildx) InspectBuilder(context.Context, string) error                   { return nil }
func (client *fakeBuildx) Build(_ context.Context, request buildx.BuildRequest) error {
	client.request = request
	return os.WriteFile(filepath.Join(request.OutputDirectory, "firmware.elf"), []byte("elf"), 0o644)
}

func TestBackendUsesOneSolveContractForLocalAndRemoteBuilders(t *testing.T) {
	for _, kind := range []builder.Kind{builder.KindLocalBuildKit, builder.KindRemoteBuildKit} {
		t.Run(string(kind), func(t *testing.T) {
			client := &fakeBuildx{}
			backend := testBackend(t, kind, client)
			plan := testPlan(t)
			reference := "registry.example/environment@sha256:" + strings.Repeat("a", 64)
			definition := builder.Definition{ID: "test", Kind: kind, BuildxBuilder: "rm-relay-test", Environments: map[string]string{"embedded-development": reference}}
			evidence, err := backend.Build(context.Background(), plan, definition)
			if err != nil {
				t.Fatal(err)
			}
			if backend.Kind() != kind || evidence.EnvironmentReference != reference || client.request.Builder != "rm-relay-test" {
				t.Fatalf("backend = %q, evidence = %#v, request = %#v", backend.Kind(), evidence, client.request)
			}
			if client.request.BuildArguments["RM_RELAY_CCACHE_ID"] != "rm-relay-ccache-"+plan.Profile.Digest {
				t.Fatalf("cache ID = %q", client.request.BuildArguments["RM_RELAY_CCACHE_ID"])
			}
			if _, err := os.Stat(filepath.Join(plan.OutputDirectory, "firmware.elf")); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestBackendRejectsMissingEnvironmentBeforeSolve(t *testing.T) {
	client := &fakeBuildx{}
	backend := testBackend(t, builder.KindLocalBuildKit, client)
	definition := builder.Definition{ID: builder.LocalID, Kind: builder.KindLocalBuildKit, BuildxBuilder: builder.LocalBuildxBuilder}
	_, err := backend.Build(context.Background(), testPlan(t), definition)
	if err == nil || !strings.Contains(err.Error(), "no mapping") {
		t.Fatalf("Build() error = %v", err)
	}
}

func TestWorkspaceFrontendDiscardsLocalBuildState(t *testing.T) {
	content := string(workspaceDockerfile)
	for _, required := range []string{"rm -rf /workspace/build /workspace/install", "RM_RELAY_OUTPUT_DIR=/rm-relay-output", "type=cache"} {
		if !strings.Contains(content, required) {
			t.Fatalf("workspace frontend missing %q", required)
		}
	}
}

func testBackend(t *testing.T, kind builder.Kind, client buildx.Client) Backend {
	t.Helper()
	workflows, err := build.NewWorkflowCatalog(cmake.Workflow{})
	if err != nil {
		t.Fatal(err)
	}
	backend, err := NewBackend(kind, client, workflows, nil)
	if err != nil {
		t.Fatal(err)
	}
	return backend
}

func testPlan(t *testing.T) build.Plan {
	t.Helper()
	root := t.TempDir()
	return build.Plan{
		Operation: build.OperationBuild, ProjectRoot: root,
		ProjectID:       "1e013e16-04a7-4fd3-9f48-bfc9178f5421",
		OutputDirectory: filepath.Join(root, "install", "embedded-test"),
		Build:           project.Build{Profile: "embedded-test", System: "cmake", Preset: "stm32", Outputs: []project.Output{{Role: "firmware.elf", Path: "firmware.elf"}}},
		Profile: profile.Loaded{Digest: strings.Repeat("b", 64), Config: profile.Config{
			ID: "embedded-test", Environment: profile.Environment{ID: "embedded-development"}, RequiredOutputRoles: []string{"firmware.elf"},
		}},
	}
}
