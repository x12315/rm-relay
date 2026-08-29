package openocd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/x12315/rm-relay/toolkit/internal/buildoutput"
	"github.com/x12315/rm-relay/toolkit/internal/commandexec"
	"github.com/x12315/rm-relay/toolkit/internal/executionplan"
	"github.com/x12315/rm-relay/toolkit/internal/profile"
	"github.com/x12315/rm-relay/toolkit/internal/project"
	"github.com/x12315/rm-relay/toolkit/internal/target"
)

func TestFlashDryRunUsesConfiguredELFAndDoesNotExecute(t *testing.T) {
	fixture := openOCDTestFixture(t)
	runner := &recordingRunner{}
	adapter := Adapter{Runner: runner, MiseBinary: "/distribution/bin/mise"}

	result, err := adapter.Flash(context.Background(), target.FlashRequest{
		BuildOutput: fixture.buildOutput,
		Profile:     fixture.profile,
		TargetName:  "openocd-stlink",
		DryRun:      true,
	})
	if err != nil {
		t.Fatalf("Flash() error = %v", err)
	}
	if result.Executed || len(runner.requests) != 0 {
		t.Fatalf("dry-run executed process: result=%+v requests=%d", result, len(runner.requests))
	}
	want := []string{
		"/distribution/bin/mise", "exec", "--", "openocd",
		"-f", fixture.boardConfig,
		"-c", "program {" + fixture.elfPath + "} verify reset exit",
	}
	if !reflect.DeepEqual(result.Command, want) {
		t.Fatalf("Command = %#v, want %#v", result.Command, want)
	}
}

func TestFlashRejectsMissingTargetCapability(t *testing.T) {
	fixture := openOCDTestFixture(t)

	_, err := (Adapter{MiseBinary: "/distribution/bin/mise"}).Flash(context.Background(), target.FlashRequest{
		BuildOutput: fixture.buildOutput,
		Profile:     fixture.profile,
		TargetName:  "missing",
		DryRun:      true,
	})
	if err == nil || !strings.Contains(err.Error(), "missing") {
		t.Fatalf("Flash() error = %v, want missing target capability", err)
	}
}

func TestFlashRejectsWrongAdapterKind(t *testing.T) {
	fixture := openOCDTestFixture(t)
	fixture.profile.Config.Targets["serial"] = profile.Target{
		Adapter:      "serial",
		Config:       "openocd/board.cfg",
		ArtifactRole: "firmware.elf",
	}

	_, err := (Adapter{MiseBinary: "/distribution/bin/mise"}).Flash(context.Background(), target.FlashRequest{
		BuildOutput: fixture.buildOutput,
		Profile:     fixture.profile,
		TargetName:  "serial",
		DryRun:      true,
	})
	if err == nil || !strings.Contains(err.Error(), "serial") {
		t.Fatalf("Flash() error = %v, want adapter kind error", err)
	}
}

func TestFlashPreservesPathsAsProcessArguments(t *testing.T) {
	fixture := openOCDTestFixture(t)
	runner := &recordingRunner{results: []commandexec.Result{{}}}
	adapter := Adapter{Runner: runner, MiseBinary: "/distribution path/bin/mise"}

	result, err := adapter.Flash(context.Background(), target.FlashRequest{
		BuildOutput: fixture.buildOutput,
		Profile:     fixture.profile,
		TargetName:  "openocd-stlink",
	})
	if err != nil {
		t.Fatalf("Flash() error = %v", err)
	}
	if !result.Executed || len(runner.requests) != 1 {
		t.Fatalf("Flash() result=%+v requests=%d", result, len(runner.requests))
	}
	request := runner.requests[0]
	if request.Name != "/distribution path/bin/mise" {
		t.Fatalf("process name = %q", request.Name)
	}
	if !reflect.DeepEqual(request.Arguments, result.Command[1:]) {
		t.Fatalf("process arguments = %#v, command = %#v", request.Arguments, result.Command)
	}
	if got := request.Environment["MISE_OVERRIDE_CONFIG_FILENAMES"]; got != filepath.Join(fixture.profile.AssetsRoot, "mise", "core.toml") {
		t.Fatalf("MISE_OVERRIDE_CONFIG_FILENAMES = %q", got)
	}
}

type openOCDTestAssets struct {
	buildOutput buildoutput.Verified
	profile     profile.Loaded
	boardConfig string
	elfPath     string
}

func openOCDTestFixture(t *testing.T) openOCDTestAssets {
	t.Helper()
	projectRoot := filepath.Join(t.TempDir(), "project with spaces")
	outputDirectory := filepath.Join(projectRoot, "install", "embedded-test")
	assetsRoot := filepath.Join(t.TempDir(), "assets with spaces")
	boardConfig := filepath.Join(assetsRoot, "openocd", "board.cfg")
	writeTestFile(t, boardConfig, "adapter speed 1800\n")
	writeTestFile(t, filepath.Join(assetsRoot, "mise", "core.toml"), "min_version = \"2026.8.14\"\n")
	elfPath := filepath.Join(outputDirectory, "firmware.elf")
	writeTestFile(t, elfPath, "elf")
	loadedProfile := profile.Loaded{
		AssetsRoot: assetsRoot,
		Digest:     strings.Repeat("a", 64),
		Config: profile.Config{
			ID:                  "embedded-test",
			DevelopmentImage:    "mcu-dev/toolchain:test",
			RequiredOutputRoles: []string{"firmware.elf"},
			Targets: map[string]profile.Target{
				"openocd-stlink": {
					Adapter:      "openocd",
					Config:       "openocd/board.cfg",
					ArtifactRole: "firmware.elf",
				},
			},
		},
	}
	plan := executionplan.Plan{
		ProjectRoot:     projectRoot,
		ProjectID:       "1e013e16-04a7-4fd3-9f48-bfc9178f5421",
		OutputDirectory: outputDirectory,
		Profile:         loadedProfile,
		Build: project.Build{
			Profile: "embedded-test",
			Outputs: []project.Output{{Role: "firmware.elf", Path: "firmware.elf"}},
		},
	}
	if _, err := buildoutput.Create(plan, "sha256:image", "0.1.0"); err != nil {
		t.Fatal(err)
	}
	verified, err := buildoutput.LoadAndValidate(outputDirectory, plan.ProjectID, loadedProfile)
	if err != nil {
		t.Fatal(err)
	}
	return openOCDTestAssets{
		buildOutput: verified,
		profile:     loadedProfile,
		boardConfig: boardConfig,
		elfPath:     elfPath,
	}
}

type recordingRunner struct {
	requests []commandexec.Request
	results  []commandexec.Result
}

func (runner *recordingRunner) Run(_ context.Context, request commandexec.Request) (commandexec.Result, error) {
	runner.requests = append(runner.requests, request)
	index := len(runner.requests) - 1
	if index >= len(runner.results) {
		return commandexec.Result{}, fmt.Errorf("unexpected process request")
	}
	return runner.results[index], nil
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
