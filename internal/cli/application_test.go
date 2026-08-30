package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/x12315/rm-relay/internal/build"
	"github.com/x12315/rm-relay/internal/profile"
	"github.com/x12315/rm-relay/internal/target"
)

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
	stdout      bytes.Buffer
	stderr      bytes.Buffer
}

func newCLIFixture(t *testing.T, projectID string) *cliFixture {
	t.Helper()
	fixture := &cliFixture{
		projectRoot: filepath.Join(t.TempDir(), "project"),
	}
	writeCLIFile(t, filepath.Join(fixture.projectRoot, "rm-relay.toml"), projectConfig(projectID))
	return fixture
}

func (fixture *cliFixture) dependencies(t *testing.T) Dependencies {
	t.Helper()
	buildBackends, err := build.NewBackendCatalog(cliTestBackend{})
	if err != nil {
		t.Fatal(err)
	}
	flashAdapters, err := target.NewFlashAdapterCatalog(cliTestFlashAdapter{})
	if err != nil {
		t.Fatal(err)
	}
	return Dependencies{
		Profiles:        profile.BuiltinCatalog(),
		BuildBackends:   buildBackends,
		DefaultBackend:  (cliTestBackend{}).ID(),
		FlashAdapters:   flashAdapters,
		ProducerVersion: "0.1.0-test",
		Stdout:          &fixture.stdout,
		Stderr:          &fixture.stderr,
	}
}

type cliTestBackend struct{}

func (cliTestBackend) ID() string { return "cli-test" }

func (cliTestBackend) Build(context.Context, build.Plan) (string, error) {
	return "", errors.New("CLI test backend must not execute")
}

type cliTestFlashAdapter struct{}

func (cliTestFlashAdapter) ID() string { return "openocd" }

func (cliTestFlashAdapter) Flash(context.Context, target.FlashRequest) (target.FlashResult, error) {
	return target.FlashResult{}, errors.New("CLI test flash adapter must not execute")
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
