package openocd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/x12315/rm-relay/internal/build/output"
	"github.com/x12315/rm-relay/internal/execution/command"
	"github.com/x12315/rm-relay/internal/execution/mise"
	"github.com/x12315/rm-relay/internal/execution/resourcecache"
	"github.com/x12315/rm-relay/internal/profile"
	"github.com/x12315/rm-relay/internal/project"
	"github.com/x12315/rm-relay/internal/target"
)

func TestFlashDryRunUsesConfiguredELFAndDoesNotExecute(t *testing.T) {
	fixture := openOCDTestFixture(t)
	runner := &recordingRunner{}
	adapter := fixture.adapter(runner, "/distribution/bin/mise")

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
	if result.Command[0] != "/distribution/bin/mise" || !containsAdjacent(result.Command, "-f", fixture.boardConfig) || !containsAdjacent(result.Command, "-c", "program {"+fixture.elfPath+"} verify reset exit") {
		t.Fatalf("Command = %#v", result.Command)
	}
}

func TestAdapterHasStableIdentity(t *testing.T) {
	if (Adapter{}).ID() != "openocd" {
		t.Fatalf("Adapter.ID() = %q", (Adapter{}).ID())
	}
}

func TestFlashRejectsMissingTargetCapability(t *testing.T) {
	fixture := openOCDTestFixture(t)

	_, err := fixture.adapter(nil, "/distribution/bin/mise").Flash(context.Background(), target.FlashRequest{
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
		Board:        "robomaster-c",
		ArtifactRole: "firmware.elf",
	}

	_, err := fixture.adapter(nil, "/distribution/bin/mise").Flash(context.Background(), target.FlashRequest{
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
	runner := &recordingRunner{results: []command.Result{{}}}
	adapter := fixture.adapter(runner, "/distribution path/bin/mise")

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
	if got := request.Environment["MISE_OVERRIDE_CONFIG_FILENAMES"]; got != fixture.miseConfig {
		t.Fatalf("MISE_OVERRIDE_CONFIG_FILENAMES = %q", got)
	}
}

type openOCDTestAssets struct {
	buildOutput output.Verified
	profile     profile.Loaded
	boards      BoardCatalog
	resources   resourcecache.Store
	boardConfig string
	miseConfig  string
	elfPath     string
}

func (fixture openOCDTestAssets) adapter(runner command.Runner, miseBinary string) Adapter {
	return Adapter{
		Runner:        runner,
		MiseBinary:    miseBinary,
		ResourceCache: fixture.resources,
		Boards:        fixture.boards,
	}
}

func openOCDTestFixture(t *testing.T) openOCDTestAssets {
	t.Helper()
	projectRoot := filepath.Join(t.TempDir(), "project with spaces")
	outputDirectory := filepath.Join(projectRoot, "install", "embedded-test")
	cacheRoot := filepath.Join(t.TempDir(), "cache with spaces")
	store := resourcecache.Store{Root: cacheRoot}
	boards := BuiltinBoardCatalog(store)
	boardConfig, err := boards.Resolve("robomaster-c")
	if err != nil {
		t.Fatal(err)
	}
	miseConfig, err := mise.MaterializeBaseConfig(store)
	if err != nil {
		t.Fatal(err)
	}
	elfPath := filepath.Join(outputDirectory, "firmware.elf")
	writeTestFile(t, elfPath, "elf")
	loadedProfile := profile.Loaded{
		Digest: strings.Repeat("a", 64),
		Config: profile.Config{
			ID:                  "embedded-test",
			DevelopmentImage:    "mcu-dev/toolchain:test",
			RequiredOutputRoles: []string{"firmware.elf"},
			Targets: map[string]profile.Target{
				"openocd-stlink": {
					Adapter:      "openocd",
					Board:        "robomaster-c",
					ArtifactRole: "firmware.elf",
				},
			},
		},
	}
	if _, err := output.Create(output.CreateRequest{
		OutputDirectory: outputDirectory,
		ProjectID:       "1e013e16-04a7-4fd3-9f48-bfc9178f5421",
		Profile:         loadedProfile,
		DeclaredOutputs: []project.Output{{Role: "firmware.elf", Path: "firmware.elf"}},
		ImageID:         "sha256:image",
		ProducerVersion: "0.1.0",
	}); err != nil {
		t.Fatal(err)
	}
	verified, err := output.LoadAndValidate(outputDirectory, "1e013e16-04a7-4fd3-9f48-bfc9178f5421", loadedProfile)
	if err != nil {
		t.Fatal(err)
	}
	return openOCDTestAssets{
		buildOutput: verified,
		profile:     loadedProfile,
		boards:      boards,
		resources:   store,
		boardConfig: boardConfig,
		miseConfig:  miseConfig,
		elfPath:     elfPath,
	}
}

type recordingRunner struct {
	requests []command.Request
	results  []command.Result
}

func (runner *recordingRunner) Run(_ context.Context, request command.Request) (command.Result, error) {
	runner.requests = append(runner.requests, request)
	index := len(runner.requests) - 1
	if index >= len(runner.results) {
		return command.Result{}, fmt.Errorf("unexpected process request")
	}
	return runner.results[index], nil
}

func containsAdjacent(arguments []string, first, second string) bool {
	for index := 0; index+1 < len(arguments); index++ {
		if arguments[index] == first && arguments[index+1] == second {
			return true
		}
	}
	return false
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
