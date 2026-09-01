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
	buildkitbackend "github.com/x12315/rm-relay/internal/build/backend/buildkit"
	"github.com/x12315/rm-relay/internal/build/cmake"
	"github.com/x12315/rm-relay/internal/build/output"
	"github.com/x12315/rm-relay/internal/builder"
	"github.com/x12315/rm-relay/internal/cli"
	"github.com/x12315/rm-relay/internal/execution/buildx"
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
	builders    *integrationBuilderManager
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
		builders:    &integrationBuilderManager{},
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
	backend, err := buildkitbackend.NewBackend(builder.KindLocalBuildKit, integrationBuildx{}, workflows, nil)
	if err != nil {
		t.Fatal(err)
	}
	backends, err := build.NewBackendCatalog(backend)
	if err != nil {
		t.Fatal(err)
	}
	reference := "registry.example/environment@sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	builders, err := builder.NewCatalog(builder.Definition{ID: builder.LocalID, Kind: builder.KindLocalBuildKit, BuildxBuilder: builder.LocalBuildxBuilder, Environments: map[string]string{"embedded-development": reference}})
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
		Builders:        builders,
		BuilderManager:  fixture.builders,
		BuildBackends:   backends,
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
	requests []command.Request
}

func (runner *recordingRunner) Run(_ context.Context, request command.Request) (command.Result, error) {
	runner.requests = append(runner.requests, request)
	return command.Result{}, fmt.Errorf("unexpected process %q", request.Name)
}

type integrationBuildx struct{}

type integrationBuilderManager struct{ prepared []string }

func (integrationBuilderManager) Add(context.Context, builder.AddRequest) error { return nil }
func (integrationBuilderManager) Remove(context.Context, string) error          { return nil }
func (integrationBuilderManager) SetEnvironment(string, string, string) error   { return nil }
func (integrationBuilderManager) List() ([]builder.Definition, error)           { return nil, nil }
func (manager *integrationBuilderManager) Prepare(_ context.Context, id string) error {
	manager.prepared = append(manager.prepared, id)
	return nil
}
func (integrationBuilderManager) Check(context.Context, string) error { return nil }

func (integrationBuildx) ListBuilders(context.Context) ([]buildx.BuilderSummary, error) {
	return nil, nil
}
func (integrationBuildx) CreateLocal(context.Context, buildx.CreateLocalRequest) error   { return nil }
func (integrationBuildx) CreateRemote(context.Context, buildx.CreateRemoteRequest) error { return nil }
func (integrationBuildx) RemoveBuilder(context.Context, string) error                    { return nil }
func (integrationBuildx) InspectBuilder(context.Context, string) error                   { return nil }
func (integrationBuildx) Build(_ context.Context, request buildx.BuildRequest) error {
	for name, contents := range map[string]string{"robomaster-c-starter.elf": "elf", "robomaster-c-starter.bin": "bin", "robomaster-c-starter.map": "map"} {
		if err := os.WriteFile(filepath.Join(request.OutputDirectory, name), []byte(contents), 0o644); err != nil {
			return err
		}
	}
	return nil
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
