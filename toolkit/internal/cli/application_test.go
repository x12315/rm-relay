package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/x12315/rm-relay/toolkit/internal/buildoutput"
	"github.com/x12315/rm-relay/toolkit/internal/commandexec"
	"github.com/x12315/rm-relay/toolkit/internal/executionplan"
)

func TestInitPrintsGeneratedProjectIdentity(t *testing.T) {
	fixture := newCLIFixture(t, "")

	exitCode := Run(context.Background(), []string{"--project", fixture.projectRoot, "init"}, fixture.dependencies())
	if exitCode != 0 {
		t.Fatalf("Run() exitCode = %d, stderr = %s", exitCode, fixture.stderr.String())
	}
	if !strings.Contains(fixture.stdout.String(), "项目标识") {
		t.Fatalf("stdout = %q", fixture.stdout.String())
	}
	projectConfig, err := os.ReadFile(filepath.Join(fixture.projectRoot, "rm-relay.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(projectConfig), `project_id = ""`) {
		t.Fatal("init left project_id blank")
	}
}

func TestBuildRunsLocalBackendAndPrintsManifestPath(t *testing.T) {
	fixture := newCLIFixture(t, testProjectID)
	fixture.runner.onDockerRun = func() {
		fixture.writeBuildArtifacts(t)
	}

	exitCode := Run(context.Background(), []string{"--project", fixture.projectRoot, "build"}, fixture.dependencies())
	if exitCode != 0 {
		t.Fatalf("Run() exitCode = %d, stderr = %s", exitCode, fixture.stderr.String())
	}
	manifestPath := filepath.Join(fixture.outputDirectory(), buildoutput.ManifestFileName)
	if _, err := os.Stat(manifestPath); err != nil {
		t.Fatalf("manifest was not created: %v", err)
	}
	if !strings.Contains(fixture.stdout.String(), filepath.ToSlash(filepath.Join("install", testProfileID, buildoutput.ManifestFileName))) {
		t.Fatalf("stdout = %q", fixture.stdout.String())
	}
	if len(fixture.runner.requests) != 2 || fixture.runner.requests[1].Name != "docker" {
		t.Fatalf("process requests = %#v", fixture.runner.requests)
	}
}

func TestFlashDryRunValidatesOutputAndPrintsOpenOCDCommand(t *testing.T) {
	fixture := newCLIFixture(t, testProjectID)
	fixture.createBuildOutput(t)

	exitCode := Run(context.Background(), []string{
		"--project", fixture.projectRoot,
		"flash", "--target", "openocd-stlink", "--dry-run",
	}, fixture.dependencies())
	if exitCode != 0 {
		t.Fatalf("Run() exitCode = %d, stderr = %s", exitCode, fixture.stderr.String())
	}
	if len(fixture.runner.requests) != 0 {
		t.Fatalf("dry-run launched processes: %#v", fixture.runner.requests)
	}
	if !strings.Contains(fixture.stdout.String(), "openocd") || !strings.Contains(fixture.stdout.String(), "verify reset exit") {
		t.Fatalf("stdout = %q", fixture.stdout.String())
	}
}

func TestJSONSuccessWritesOneObjectToStdout(t *testing.T) {
	fixture := newCLIFixture(t, testProjectID)
	fixture.runner.onDockerRun = func() {
		fixture.writeBuildArtifacts(t)
	}

	exitCode := Run(context.Background(), []string{
		"--project", fixture.projectRoot,
		"--format", "json",
		"build",
	}, fixture.dependencies())
	if exitCode != 0 {
		t.Fatalf("Run() exitCode = %d, stderr = %s", exitCode, fixture.stderr.String())
	}
	decoder := json.NewDecoder(strings.NewReader(fixture.stdout.String()))
	var result map[string]any
	if err := decoder.Decode(&result); err != nil {
		t.Fatalf("decode stdout: %v\n%s", err, fixture.stdout.String())
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		t.Fatalf("stdout contains trailing JSON data: %v\n%s", err, fixture.stdout.String())
	}
	if result["ok"] != true || result["operation"] != "build" || result["profile"] != testProfileID {
		t.Fatalf("JSON result = %#v", result)
	}
}

func TestJSONFailureUsesStableErrorCodeAndNonZeroExit(t *testing.T) {
	fixture := newCLIFixture(t, testProjectID)
	if err := os.WriteFile(filepath.Join(fixture.projectRoot, "rm-relay.toml"), []byte("schema_version = 99\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	exitCode := Run(context.Background(), []string{
		"--project", fixture.projectRoot,
		"--format", "json",
		"build",
	}, fixture.dependencies())
	if exitCode == 0 {
		t.Fatal("Run() succeeded for an invalid project")
	}
	var result struct {
		OK    bool `json:"ok"`
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(fixture.stdout.Bytes(), &result); err != nil {
		t.Fatalf("decode stdout: %v\n%s", err, fixture.stdout.String())
	}
	if result.OK || result.Error.Code != "project_invalid" {
		t.Fatalf("JSON error = %#v", result)
	}
}

const (
	testProjectID = "1e013e16-04a7-4fd3-9f48-bfc9178f5421"
	testProfileID = "embedded-stm32f407-robomaster-c"
)

type cliFixture struct {
	projectRoot string
	homeRoot    string
	assetsRoot  string
	runner      *cliRecordingRunner
	stdout      bytes.Buffer
	stderr      bytes.Buffer
}

func newCLIFixture(t *testing.T, projectID string) *cliFixture {
	t.Helper()
	fixture := &cliFixture{
		projectRoot: filepath.Join(t.TempDir(), "project"),
		homeRoot:    filepath.Join(t.TempDir(), "distribution"),
		runner:      &cliRecordingRunner{},
	}
	fixture.assetsRoot = filepath.Join(fixture.homeRoot, "share", "rm-relay")
	writeCLIFile(t, filepath.Join(fixture.projectRoot, "rm-relay.toml"), projectConfig(projectID))
	writeCLIFile(t, filepath.Join(fixture.projectRoot, "mise.toml"), "min_version = \"2026.8.14\"\n")
	writeCLIFile(t, filepath.Join(fixture.assetsRoot, "mise", "core.toml"), "min_version = \"2026.8.14\"\n")
	writeCLIFile(t, filepath.Join(fixture.assetsRoot, "profiles", testProfileID, "mise.toml"), "min_version = \"2026.8.14\"\n")
	writeCLIFile(t, filepath.Join(fixture.assetsRoot, "profiles", testProfileID, "profile.toml"), profileConfig())
	writeCLIFile(t, filepath.Join(fixture.assetsRoot, "openocd", "boards", "robomaster-c.cfg"), "adapter speed 1800\n")
	return fixture
}

func (fixture *cliFixture) dependencies() Dependencies {
	return Dependencies{
		Runner:          fixture.runner,
		HomeDirectory:   fixture.homeRoot,
		MiseBinary:      filepath.Join(fixture.homeRoot, "bin", "mise"),
		ProducerVersion: "0.1.0-test",
		Stdout:          &fixture.stdout,
		Stderr:          &fixture.stderr,
	}
}

func (fixture *cliFixture) outputDirectory() string {
	return filepath.Join(fixture.projectRoot, "install", testProfileID)
}

func (fixture *cliFixture) writeBuildArtifacts(t *testing.T) {
	t.Helper()
	writeCLIFile(t, filepath.Join(fixture.outputDirectory(), "firmware.elf"), "elf")
	writeCLIFile(t, filepath.Join(fixture.outputDirectory(), "firmware.bin"), "bin")
	writeCLIFile(t, filepath.Join(fixture.outputDirectory(), "linker.map"), "map")
}

func (fixture *cliFixture) createBuildOutput(t *testing.T) {
	t.Helper()
	fixture.writeBuildArtifacts(t)
	plan, err := executionplan.Resolve(executionplan.OperationBuild, fixture.projectRoot, fixture.assetsRoot, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := buildoutput.Create(plan, "sha256:image", "0.1.0-test"); err != nil {
		t.Fatal(err)
	}
}

type cliRecordingRunner struct {
	requests    []commandexec.Request
	onDockerRun func()
}

func (runner *cliRecordingRunner) Run(_ context.Context, request commandexec.Request) (commandexec.Result, error) {
	runner.requests = append(runner.requests, request)
	if isDockerImageInspect(request) {
		return commandexec.Result{Stdout: "sha256:image\n"}, nil
	}
	if request.Name == "docker" && len(request.Arguments) > 0 && request.Arguments[0] == "run" {
		if runner.onDockerRun != nil {
			runner.onDockerRun()
		}
		return commandexec.Result{}, nil
	}
	return commandexec.Result{}, nil
}

func isDockerImageInspect(request commandexec.Request) bool {
	return request.Name == "docker" && len(request.Arguments) >= 2 && request.Arguments[0] == "image" && request.Arguments[1] == "inspect"
}

func projectConfig(projectID string) string {
	return `schema_version = 1
project_id = "` + projectID + `"
default_profile = "` + testProfileID + `"

[[builds]]
profile = "` + testProfileID + `"
mise_config = "mise.toml"
task = "rm-relay:build:stm32f407-robomaster-c"

[[builds.outputs]]
role = "firmware.elf"
path = "firmware.elf"

[[builds.outputs]]
role = "firmware.bin"
path = "firmware.bin"

[[builds.outputs]]
role = "linker.map"
path = "linker.map"
`
}

func profileConfig() string {
	return `schema_version = 1
id = "` + testProfileID + `"
development_image = "mcu-dev/toolchain:test"
mise_config = "mise.toml"
required_output_roles = ["firmware.elf", "firmware.bin", "linker.map"]

[targets.openocd-stlink]
adapter = "openocd"
config = "openocd/boards/robomaster-c.cfg"
artifact_role = "firmware.elf"
`
}

func writeCLIFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
