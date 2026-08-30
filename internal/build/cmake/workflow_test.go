package cmake

import (
	"os"
	"reflect"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/x12315/rm-relay/internal/build"
	"github.com/x12315/rm-relay/internal/execution/mise"
	"github.com/x12315/rm-relay/internal/profile"
	"github.com/x12315/rm-relay/internal/project"
)

func TestInternalBuildTaskRunsFromCallerWorkspace(t *testing.T) {
	contents, err := os.ReadFile("build.mise.toml")
	if err != nil {
		t.Fatal(err)
	}
	var config struct {
		Tasks map[string]struct {
			Directory string `toml:"dir"`
		} `toml:"tasks"`
	}
	if _, err := toml.Decode(string(contents), &config); err != nil {
		t.Fatalf("decode build.mise.toml: %v", err)
	}

	task, exists := config.Tasks[workspaceTaskName]
	if !exists {
		t.Fatalf("task %q is not defined", workspaceTaskName)
	}
	if task.Directory != "{{cwd}}" {
		t.Fatalf("task directory = %q, want caller workspace", task.Directory)
	}
}

func TestPrepareUsesInternalConfigsAndProjectPreset(t *testing.T) {
	plan := build.Plan{
		Build: project.Build{System: System, Preset: "stm32f407-robomaster-c"},
		Profile: profile.Loaded{Config: profile.Config{
			ID: "embedded-stm32f407-robomaster-c",
		}},
	}

	task, err := (Workflow{}).Prepare(plan)
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
	if !reflect.DeepEqual(task.MiseConfigFiles, []string{mise.ContainerBaseConfig, containerConfigPath}) {
		t.Fatalf("MiseConfigFiles = %#v", task.MiseConfigFiles)
	}
	if task.Environment["RM_RELAY_BUILD_PRESET"] != "stm32f407-robomaster-c" {
		t.Fatalf("preset environment = %q", task.Environment["RM_RELAY_BUILD_PRESET"])
	}
}
