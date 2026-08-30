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

	"github.com/x12315/rm-relay/internal/build"
	"github.com/x12315/rm-relay/internal/build/backend/localcontainer"
	"github.com/x12315/rm-relay/internal/build/cmake"
	"github.com/x12315/rm-relay/internal/build/output"
	"github.com/x12315/rm-relay/internal/execution/command"
	"github.com/x12315/rm-relay/internal/execution/resourcecache"
	"github.com/x12315/rm-relay/internal/profile"
	"github.com/x12315/rm-relay/internal/target"
	openocdtarget "github.com/x12315/rm-relay/internal/target/openocd"
)

func TestInitPrintsGeneratedProjectIdentity(t *testing.T) {
	fixture := newCLIFixture(t, "")

	exitCode := Run(context.Background(), []string{"--project", fixture.projectRoot, "init"}, fixture.dependencies(t))
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
	fixture.runner.onDockerRun = func() { fixture.writeBuildArtifacts(t) }

	exitCode := Run(context.Background(), []string{"build", "--project", fixture.projectRoot}, fixture.dependencies(t))
	if exitCode != 0 {
		t.Fatalf("Run() exitCode = %d, stderr = %s", exitCode, fixture.stderr.String())
	}
	manifestPath := filepath.Join(fixture.outputDirectory(), output.ManifestFileName)
	if _, err := os.Stat(manifestPath); err != nil {
		t.Fatalf("manifest was not created: %v", err)
	}
	if !strings.Contains(fixture.stdout.String(), filepath.ToSlash(filepath.Join("install", testProfileID, output.ManifestFileName))) {
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
		"flash", "--project", fixture.projectRoot,
		"--target", "openocd-stlink", "--dry-run",
	}, fixture.dependencies(t))
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
	fixture.runner.onDockerRun = func() { fixture.writeBuildArtifacts(t) }

	exitCode := Run(context.Background(), []string{
		"build", "--project", fixture.projectRoot, "--format", "json",
	}, fixture.dependencies(t))
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
		"build", "--project", fixture.projectRoot, "--format", "json",
	}, fixture.dependencies(t))
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

func TestJSONArgumentFailureIdentifiesTheInvokedOperation(t *testing.T) {
	fixture := newCLIFixture(t, testProjectID)

	exitCode := Run(context.Background(), []string{
		"build", "unexpected", "--format", "json",
	}, fixture.dependencies(t))
	if exitCode != 2 {
		t.Fatalf("Run() exitCode = %d, want 2", exitCode)
	}
	var result failureResult
	if err := json.Unmarshal(fixture.stdout.Bytes(), &result); err != nil {
		t.Fatalf("decode stdout: %v\n%s", err, fixture.stdout.String())
	}
	if result.Operation != "build" || result.Error.Code != "invalid_arguments" {
		t.Fatalf("JSON error = %#v", result)
	}
}

func TestHelpDescribesStableCommandsWithoutExposingMise(t *testing.T) {
	fixture := newCLIFixture(t, testProjectID)
	exitCode := Run(context.Background(), []string{"--help"}, fixture.dependencies(t))
	if exitCode != 0 {
		t.Fatalf("Run() exitCode = %d", exitCode)
	}
	help := fixture.stdout.String()
	for _, commandName := range []string{"init", "build", "flash", "completion"} {
		if !strings.Contains(help, commandName) {
			t.Fatalf("help does not contain %q:\n%s", commandName, help)
		}
	}
	if strings.Contains(strings.ToLower(help), "mise") {
		t.Fatalf("help exposes internal mise layer:\n%s", help)
	}
}

func TestCompletionSupportsZsh(t *testing.T) {
	fixture := newCLIFixture(t, testProjectID)
	exitCode := Run(context.Background(), []string{"completion", "zsh"}, fixture.dependencies(t))
	if exitCode != 0 {
		t.Fatalf("Run() exitCode = %d, stderr = %s", exitCode, fixture.stderr.String())
	}
	if !strings.Contains(fixture.stdout.String(), "compdef") {
		t.Fatalf("completion output = %q", fixture.stdout.String())
	}
}

const (
	testProjectID = "1e013e16-04a7-4fd3-9f48-bfc9178f5421"
	testProfileID = "embedded-stm32f407-robomaster-c"
)

type cliFixture struct {
	projectRoot string
	cacheRoot   string
	runner      *cliRecordingRunner
	stdout      bytes.Buffer
	stderr      bytes.Buffer
}

func newCLIFixture(t *testing.T, projectID string) *cliFixture {
	t.Helper()
	fixture := &cliFixture{
		projectRoot: filepath.Join(t.TempDir(), "project"),
		cacheRoot:   filepath.Join(t.TempDir(), "cache"),
		runner:      &cliRecordingRunner{},
	}
	writeCLIFile(t, filepath.Join(fixture.projectRoot, "rm-relay.toml"), projectConfig(projectID))
	return fixture
}

func (fixture *cliFixture) dependencies(t *testing.T) Dependencies {
	t.Helper()
	workflows, err := build.NewWorkflowCatalog(cmake.Workflow{})
	if err != nil {
		t.Fatal(err)
	}
	store := resourcecache.Store{Root: fixture.cacheRoot}
	buildBackends, err := build.NewBackendCatalog(localcontainer.Backend{
		Runner:         fixture.runner,
		Workflows:      workflows,
		CacheDirectory: filepath.Join(store.Root, "build", localcontainer.ID),
		Progress:       &fixture.stderr,
	})
	if err != nil {
		t.Fatal(err)
	}
	flashAdapters, err := target.NewFlashAdapterCatalog(openocdtarget.Adapter{
		Runner:        fixture.runner,
		MiseBinary:    filepath.Join("distribution", "bin", "mise"),
		ResourceCache: store,
		Boards:        openocdtarget.BuiltinBoardCatalog(store),
		Progress:      &fixture.stderr,
	})
	if err != nil {
		t.Fatal(err)
	}
	return Dependencies{
		Profiles:        profile.BuiltinCatalog(),
		BuildBackends:   buildBackends,
		DefaultBackend:  localcontainer.ID,
		FlashAdapters:   flashAdapters,
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
	writeCLIFile(t, filepath.Join(fixture.outputDirectory(), "robomaster-c-starter.elf"), "elf")
	writeCLIFile(t, filepath.Join(fixture.outputDirectory(), "robomaster-c-starter.bin"), "bin")
	writeCLIFile(t, filepath.Join(fixture.outputDirectory(), "robomaster-c-starter.map"), "map")
}

func (fixture *cliFixture) createBuildOutput(t *testing.T) {
	t.Helper()
	fixture.writeBuildArtifacts(t)
	plan, err := build.Resolve(build.OperationBuild, fixture.projectRoot, "", profile.BuiltinCatalog())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := output.Create(output.CreateRequest{
		OutputDirectory: plan.OutputDirectory,
		ProjectID:       plan.ProjectID,
		Profile:         plan.Profile,
		DeclaredOutputs: plan.Build.Outputs,
		ImageID:         "sha256:image",
		ProducerVersion: "0.1.0-test",
	}); err != nil {
		t.Fatal(err)
	}
}

type cliRecordingRunner struct {
	requests    []command.Request
	onDockerRun func()
}

func (runner *cliRecordingRunner) Run(_ context.Context, request command.Request) (command.Result, error) {
	runner.requests = append(runner.requests, request)
	if isDockerImageInspect(request) {
		return command.Result{Stdout: "sha256:image\n"}, nil
	}
	if request.Name == "docker" && len(request.Arguments) > 0 && request.Arguments[0] == "run" {
		if runner.onDockerRun != nil {
			runner.onDockerRun()
		}
		return command.Result{}, nil
	}
	return command.Result{}, nil
}

func isDockerImageInspect(request command.Request) bool {
	return request.Name == "docker" && len(request.Arguments) >= 2 && request.Arguments[0] == "image" && request.Arguments[1] == "inspect"
}

func projectConfig(projectID string) string {
	return `schema_version = 1
project_id = "` + projectID + `"
default_profile = "` + testProfileID + `"

[[builds]]
profile = "` + testProfileID + `"
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
