package localcontainer

import (
	"context"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/x12315/rm-relay/internal/build"
	"github.com/x12315/rm-relay/internal/build/cmake"
	"github.com/x12315/rm-relay/internal/builder"
	"github.com/x12315/rm-relay/internal/execution/command"
	"github.com/x12315/rm-relay/internal/execution/docker"
	"github.com/x12315/rm-relay/internal/profile"
	"github.com/x12315/rm-relay/internal/project"
)

func TestBackendInspectsImageBeforeStartingBuilder(t *testing.T) {
	runner := &recordingRunner{results: []command.Result{{Stdout: "sha256:image\n"}, {}}}
	backend := testBackend(t, runner)

	evidence, err := backend.Build(context.Background(), localPlan(t), localBuilder())
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if evidence.EnvironmentDigest != "sha256:image" {
		t.Fatalf("environment digest = %q", evidence.EnvironmentDigest)
	}
	if len(runner.requests) != 2 {
		t.Fatalf("requests = %d, want image inspect and docker run", len(runner.requests))
	}
	wantInspect := []string{"image", "inspect", "--format", "{{.Id}}", "mcu-dev/toolchain:test"}
	if runner.requests[0].Name != "docker" || !reflect.DeepEqual(runner.requests[0].Arguments, wantInspect) {
		t.Fatalf("inspect request = %#v", runner.requests[0])
	}
}

func TestBackendHasStableIdentity(t *testing.T) {
	if (Backend{}).Kind() != builder.KindLocalContainer {
		t.Fatalf("Backend.Kind() = %q", (Backend{}).Kind())
	}
}

func TestBackendMountsOnlyProjectAndExpendableBuildCache(t *testing.T) {
	runner := &recordingRunner{results: []command.Result{{Stdout: "sha256:image\n"}, {}}}
	backend := testBackend(t, runner)
	plan := localPlan(t)

	if _, err := backend.Build(context.Background(), plan, localBuilder()); err != nil {
		t.Fatal(err)
	}
	arguments := runner.requests[1].Arguments
	assertAdjacentArguments(t, arguments, "--volume", plan.ProjectRoot+":"+containerProjectRoot)
	assertAdjacentArguments(t, arguments, "--volume", backend.CacheDirectory+":"+containerCacheRoot)
	assertAdjacentArguments(t, arguments, "--workdir", containerProjectRoot)
	if strings.Contains(strings.Join(arguments, "\n"), "share/rm-relay") {
		t.Fatalf("backend still mounts a global asset tree:\n%s", strings.Join(arguments, "\n"))
	}
}

func TestBackendInvokesInternalWorkflowWithoutProjectMiseConfig(t *testing.T) {
	runner := &recordingRunner{results: []command.Result{{Stdout: "sha256:image\n"}, {}}}
	backend := testBackend(t, runner)
	plan := localPlan(t)

	if _, err := backend.Build(context.Background(), plan, localBuilder()); err != nil {
		t.Fatal(err)
	}
	arguments := runner.requests[1].Arguments
	wantSuffix := []string{"mcu-dev/toolchain:test", "mise", "--locked", "run", "rm-relay:build"}
	if !reflect.DeepEqual(arguments[len(arguments)-len(wantSuffix):], wantSuffix) {
		t.Fatalf("docker run suffix = %#v, want %#v", arguments[len(arguments)-len(wantSuffix):], wantSuffix)
	}
	assertAdjacentArguments(t, arguments, "--env", "MISE_OVERRIDE_CONFIG_FILENAMES=/opt/rm-relay/execution/mise/base.mise.toml:/opt/rm-relay/build/cmake/build.mise.toml")
	assertAdjacentArguments(t, arguments, "--env", "MISE_TASK_RUN_AUTO_INSTALL=false")
	assertAdjacentArguments(t, arguments, "--env", "RM_RELAY_BUILD_PRESET=stm32f407-robomaster-c")
	assertAdjacentArguments(t, arguments, "--env", "RM_RELAY_OUTPUT_DIR=/workspace/install/embedded-test")
	assertAdjacentArguments(t, arguments, "--env", "CCACHE_DIR=/cache/ccache")
}

func TestBackendRejectsUnknownBuildSystemBeforeUsingDocker(t *testing.T) {
	runner := &recordingRunner{}
	backend := testBackend(t, runner)
	plan := localPlan(t)
	plan.Build.System = "unknown"

	_, err := backend.Build(context.Background(), plan, localBuilder())
	if err == nil || !strings.Contains(err.Error(), "unsupported build system") {
		t.Fatalf("Build() error = %v", err)
	}
	if len(runner.requests) != 0 {
		t.Fatalf("Docker was used for an invalid plan: %#v", runner.requests)
	}
}

type recordingRunner struct {
	requests []command.Request
	results  []command.Result
	err      error
}

func (runner *recordingRunner) Run(_ context.Context, request command.Request) (command.Result, error) {
	runner.requests = append(runner.requests, request)
	if runner.err != nil {
		return command.Result{}, runner.err
	}
	index := len(runner.requests) - 1
	if index >= len(runner.results) {
		return command.Result{}, fmt.Errorf("unexpected request %d", index)
	}
	return runner.results[index], nil
}

func testBackend(t *testing.T, runner command.Runner) Backend {
	t.Helper()
	workflows, err := build.NewWorkflowCatalog(cmake.Workflow{})
	if err != nil {
		t.Fatal(err)
	}
	return Backend{Docker: docker.CLI{Runner: runner}, Workflows: workflows, CacheDirectory: t.TempDir()}
}

func localPlan(t *testing.T) build.Plan {
	t.Helper()
	projectRoot := t.TempDir()
	return build.Plan{
		Operation:       build.OperationBuild,
		ProjectRoot:     projectRoot,
		ProjectID:       "1e013e16-04a7-4fd3-9f48-bfc9178f5421",
		OutputDirectory: filepath.Join(projectRoot, "install", "embedded-test"),
		Build: project.Build{
			Profile: "embedded-test",
			System:  "cmake",
			Preset:  "stm32f407-robomaster-c",
		},
		Profile: profile.Loaded{Config: profile.Config{
			ID:          "embedded-test",
			Environment: profile.Environment{ID: "embedded-development", LocalReference: "mcu-dev/toolchain:test"},
		}},
	}
}

func localBuilder() builder.Definition {
	return builder.Definition{ID: builder.LocalID, Kind: builder.KindLocalContainer}
}

func assertAdjacentArguments(t *testing.T, arguments []string, first, second string) {
	t.Helper()
	for index := 0; index+1 < len(arguments); index++ {
		if arguments[index] == first && arguments[index+1] == second {
			return
		}
	}
	t.Fatalf("arguments do not contain %q followed by %q:\n%s", first, second, strings.Join(arguments, "\n"))
}
