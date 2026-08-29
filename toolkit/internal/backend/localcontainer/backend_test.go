package localcontainer

import (
	"context"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/x12315/rm-relay/toolkit/internal/commandexec"
	"github.com/x12315/rm-relay/toolkit/internal/executionplan"
	"github.com/x12315/rm-relay/toolkit/internal/profile"
	"github.com/x12315/rm-relay/toolkit/internal/project"
)

func TestBackendInspectsImageBeforeStartingBuilder(t *testing.T) {
	runner := &recordingRunner{results: []commandexec.Result{{Stdout: "sha256:image\n"}, {}}}
	backend := Backend{Runner: runner}

	imageID, err := backend.Build(context.Background(), localPlan(t))
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if imageID != "sha256:image" {
		t.Fatalf("imageID = %q", imageID)
	}
	if len(runner.requests) != 2 {
		t.Fatalf("requests = %d, want image inspect and docker run", len(runner.requests))
	}
	wantInspect := []string{"image", "inspect", "--format", "{{.Id}}", "mcu-dev/toolchain:test"}
	if runner.requests[0].Name != "docker" || !reflect.DeepEqual(runner.requests[0].Arguments, wantInspect) {
		t.Fatalf("inspect request = %#v", runner.requests[0])
	}
	if runner.requests[1].Arguments[0] != "run" {
		t.Fatalf("second request = %#v, want docker run", runner.requests[1])
	}
}

func TestBackendMountsProjectWritableAndAssetsReadOnly(t *testing.T) {
	runner := &recordingRunner{results: []commandexec.Result{{Stdout: "sha256:image\n"}, {}}}
	plan := localPlan(t)

	if _, err := (Backend{Runner: runner}).Build(context.Background(), plan); err != nil {
		t.Fatal(err)
	}
	arguments := runner.requests[1].Arguments
	assertAdjacentArguments(t, arguments, "--volume", plan.ProjectRoot+":/workspace")
	assertAdjacentArguments(t, arguments, "--volume", plan.AssetsRoot+":/opt/rm-relay:ro")
	assertAdjacentArguments(t, arguments, "--workdir", "/workspace")
}

func TestBackendInvokesOnlyDeclaredMiseTask(t *testing.T) {
	runner := &recordingRunner{results: []commandexec.Result{{Stdout: "sha256:image\n"}, {}}}
	plan := localPlan(t)

	if _, err := (Backend{Runner: runner}).Build(context.Background(), plan); err != nil {
		t.Fatal(err)
	}
	arguments := runner.requests[1].Arguments
	wantSuffix := []string{"mcu-dev/toolchain:test", "mise", "--locked", "run", "rm-relay:build:firmware"}
	if !reflect.DeepEqual(arguments[len(arguments)-len(wantSuffix):], wantSuffix) {
		t.Fatalf("docker run suffix = %#v, want %#v", arguments[len(arguments)-len(wantSuffix):], wantSuffix)
	}
	assertAdjacentArguments(t, arguments, "--env", "MISE_OVERRIDE_CONFIG_FILENAMES=/opt/rm-relay/mise/core.toml:/opt/rm-relay/profiles/embedded-test/mise.toml:/workspace/mise.toml")
	assertAdjacentArguments(t, arguments, "--env", "MISE_TASK_RUN_AUTO_INSTALL=false")
	assertAdjacentArguments(t, arguments, "--env", "RM_RELAY_OUTPUT_DIR=/workspace/install/embedded-test")
}

type recordingRunner struct {
	requests []commandexec.Request
	results  []commandexec.Result
	err      error
}

func (runner *recordingRunner) Run(_ context.Context, request commandexec.Request) (commandexec.Result, error) {
	runner.requests = append(runner.requests, request)
	if runner.err != nil {
		return commandexec.Result{}, runner.err
	}
	index := len(runner.requests) - 1
	if index >= len(runner.results) {
		return commandexec.Result{}, fmt.Errorf("unexpected request %d", index)
	}
	return runner.results[index], nil
}

func localPlan(t *testing.T) executionplan.Plan {
	t.Helper()
	projectRoot := t.TempDir()
	assetsRoot := t.TempDir()
	return executionplan.Plan{
		Operation:         executionplan.OperationBuild,
		ProjectRoot:       projectRoot,
		ProjectID:         "1e013e16-04a7-4fd3-9f48-bfc9178f5421",
		AssetsRoot:        assetsRoot,
		Backend:           "local",
		OutputDirectory:   filepath.Join(projectRoot, "install", "embedded-test"),
		CoreMiseConfig:    filepath.Join(assetsRoot, "mise", "core.toml"),
		ProfileMiseConfig: filepath.Join(assetsRoot, "profiles", "embedded-test", "mise.toml"),
		ProjectMiseConfig: filepath.Join(projectRoot, "mise.toml"),
		Build: project.Build{
			Profile:    "embedded-test",
			MiseConfig: "mise.toml",
			Task:       "rm-relay:build:firmware",
		},
		Profile: profile.Loaded{
			Directory: filepath.Join(assetsRoot, "profiles", "embedded-test"),
			Config: profile.Config{
				ID:               "embedded-test",
				DevelopmentImage: "mcu-dev/toolchain:test",
				MiseConfig:       "mise.toml",
			},
		},
	}
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
