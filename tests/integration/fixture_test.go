package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/x12315/rm-relay/internal/build"
	"github.com/x12315/rm-relay/internal/build/backend/localcontainer"
	"github.com/x12315/rm-relay/internal/build/cmake"
	"github.com/x12315/rm-relay/internal/build/output"
	"github.com/x12315/rm-relay/internal/cli"
	"github.com/x12315/rm-relay/internal/execution/command"
	"github.com/x12315/rm-relay/internal/execution/resourcecache"
	"github.com/x12315/rm-relay/internal/profile"
	"github.com/x12315/rm-relay/internal/target"
	"github.com/x12315/rm-relay/internal/target/openocd"
)

const (
	testProfileID       = "embedded-stm32f407-robomaster-c"
	testProducerVersion = "0.1.0-integration"
)

type developmentCycleFixture struct {
	projectRoot string
	cacheRoot   string
	runner      *recordingRunner
}

type cliResult struct {
	exitCode int
	stdout   string
	stderr   string
}

func newDevelopmentCycleFixture(t *testing.T) *developmentCycleFixture {
	t.Helper()
	fixture := &developmentCycleFixture{
		projectRoot: filepath.Join(t.TempDir(), "project"),
		cacheRoot:   filepath.Join(t.TempDir(), "cache"),
		runner:      &recordingRunner{},
	}
	writeFile(t, filepath.Join(fixture.projectRoot, "rm-relay.toml"), projectConfig)
	return fixture
}

func (fixture *developmentCycleFixture) runCLI(t *testing.T, arguments ...string) cliResult {
	t.Helper()
	workflows, err := build.NewWorkflowCatalog(cmake.Workflow{})
	if err != nil {
		t.Fatal(err)
	}
	store := resourcecache.Store{Root: fixture.cacheRoot}
	backends, err := build.NewBackendCatalog(localcontainer.Backend{
		Runner:         fixture.runner,
		Workflows:      workflows,
		CacheDirectory: filepath.Join(fixture.cacheRoot, "build", localcontainer.ID),
	})
	if err != nil {
		t.Fatal(err)
	}
	adapters, err := target.NewFlashAdapterCatalog(openocd.Adapter{
		Runner:        fixture.runner,
		MiseBinary:    "mise",
		ResourceCache: store,
		Boards:        openocd.BuiltinBoardCatalog(store),
	})
	if err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := cli.Run(context.Background(), append([]string{"--project", fixture.projectRoot}, arguments...), cli.Dependencies{
		Profiles:        profile.BuiltinCatalog(),
		BuildBackends:   backends,
		DefaultBackend:  localcontainer.ID,
		FlashAdapters:   adapters,
		ProducerVersion: testProducerVersion,
		Stdout:          &stdout,
		Stderr:          &stderr,
	})
	return cliResult{exitCode: exitCode, stdout: stdout.String(), stderr: stderr.String()}
}

func (fixture *developmentCycleFixture) outputDirectory() string {
	return filepath.Join(fixture.projectRoot, "install", testProfileID)
}

func (fixture *developmentCycleFixture) writeInstalledArtifacts(t *testing.T) {
	t.Helper()
	writeFile(t, filepath.Join(fixture.outputDirectory(), "robomaster-c-starter.elf"), "elf")
	writeFile(t, filepath.Join(fixture.outputDirectory(), "robomaster-c-starter.bin"), "bin")
	writeFile(t, filepath.Join(fixture.outputDirectory(), "robomaster-c-starter.map"), "map")
}

type recordingRunner struct {
	requests    []command.Request
	onDockerRun func()
}

func (runner *recordingRunner) Run(_ context.Context, request command.Request) (command.Result, error) {
	runner.requests = append(runner.requests, request)
	if request.Name == "docker" && len(request.Arguments) >= 2 && request.Arguments[0] == "image" && request.Arguments[1] == "inspect" {
		return command.Result{Stdout: "sha256:integration-image\n"}, nil
	}
	if request.Name == "docker" && len(request.Arguments) > 0 && request.Arguments[0] == "run" {
		if runner.onDockerRun != nil {
			runner.onDockerRun()
		}
		return command.Result{}, nil
	}
	return command.Result{}, fmt.Errorf("unexpected process %q", request.Name)
}

func readBuildOutputManifest(t *testing.T, path string) output.Manifest {
	t.Helper()
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var manifest output.Manifest
	if err := json.Unmarshal(contents, &manifest); err != nil {
		t.Fatalf("decode Build Output manifest: %v", err)
	}
	return manifest
}

func writeFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

const projectConfig = `schema_version = 1
project_id = ""
default_profile = "embedded-stm32f407-robomaster-c"

[[builds]]
profile = "embedded-stm32f407-robomaster-c"
system = "cmake"
preset = "stm32f407-robomaster-c"

[[builds.outputs]]
role = "firmware.elf"
path = "robomaster-c-starter.elf"

[[builds.outputs]]
role = "firmware.bin"
path = "robomaster-c-starter.bin"

[[builds.outputs]]
role = "linker.map"
path = "robomaster-c-starter.map"
`
