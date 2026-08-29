package executionplan

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveUsesProjectDefaultProfile(t *testing.T) {
	projectRoot, assetsRoot := writePlanFixture(t)

	plan, err := Resolve(OperationBuild, projectRoot, assetsRoot, "")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if plan.Profile.Config.ID != "embedded-default" {
		t.Fatalf("Profile.ID = %q", plan.Profile.Config.ID)
	}
	if plan.Backend != "local" {
		t.Fatalf("Backend = %q", plan.Backend)
	}
	wantOutput := filepath.Join(projectRoot, "install", "embedded-default")
	if plan.OutputDirectory != wantOutput {
		t.Fatalf("OutputDirectory = %q, want %q", plan.OutputDirectory, wantOutput)
	}
}

func TestResolveUsesExplicitProfileOverride(t *testing.T) {
	projectRoot, assetsRoot := writePlanFixture(t)

	plan, err := Resolve(OperationFlash, projectRoot, assetsRoot, "embedded-override")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if plan.Operation != OperationFlash {
		t.Fatalf("Operation = %q", plan.Operation)
	}
	if plan.Profile.Config.ID != "embedded-override" || plan.Build.Task != "build:override" {
		t.Fatalf("resolved override = profile %q, task %q", plan.Profile.Config.ID, plan.Build.Task)
	}
}

func TestResolveUsesThreeControlledMiseConfigs(t *testing.T) {
	projectRoot, assetsRoot := writePlanFixture(t)

	plan, err := Resolve(OperationBuild, projectRoot, assetsRoot, "")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	want := []string{
		filepath.Join(assetsRoot, "mise", "core.toml"),
		filepath.Join(assetsRoot, "profiles", "embedded-default", "profile-mise.toml"),
		filepath.Join(projectRoot, "project-mise.toml"),
	}
	actual := plan.MiseConfigs()
	for index := range want {
		if actual[index] != want[index] {
			t.Fatalf("MiseConfigs()[%d] = %q, want %q", index, actual[index], want[index])
		}
	}
}

func writePlanFixture(t *testing.T) (string, string) {
	t.Helper()
	projectRoot := t.TempDir()
	assetsRoot := t.TempDir()
	projectConfig := `schema_version = 1
project_id = "1e013e16-04a7-4fd3-9f48-bfc9178f5421"
default_profile = "embedded-default"

[[builds]]
profile = "embedded-default"
mise_config = "project-mise.toml"
task = "build:default"

[[builds.outputs]]
role = "firmware.elf"
path = "default.elf"

[[builds]]
profile = "embedded-override"
mise_config = "project-mise.toml"
task = "build:override"

[[builds.outputs]]
role = "firmware.elf"
path = "override.elf"
`
	writePlanFile(t, filepath.Join(projectRoot, "rm-relay.toml"), projectConfig)
	writePlanFile(t, filepath.Join(projectRoot, "project-mise.toml"), "min_version = \"2026.8.14\"\n")
	writePlanFile(t, filepath.Join(assetsRoot, "mise", "core.toml"), "min_version = \"2026.8.14\"\n")
	writePlanFile(t, filepath.Join(assetsRoot, "openocd", "board.cfg"), "adapter speed 1800\n")
	for _, profileID := range []string{"embedded-default", "embedded-override"} {
		profileConfig := `schema_version = 1
id = "` + profileID + `"
development_image = "mcu-dev/toolchain:test"
mise_config = "profile-mise.toml"
required_output_roles = ["firmware.elf"]

[targets.openocd-stlink]
adapter = "openocd"
config = "openocd/board.cfg"
artifact_role = "firmware.elf"
`
		profileRoot := filepath.Join(assetsRoot, "profiles", profileID)
		writePlanFile(t, filepath.Join(profileRoot, "profile.toml"), profileConfig)
		writePlanFile(t, filepath.Join(profileRoot, "profile-mise.toml"), "min_version = \"2026.8.14\"\n")
	}
	return projectRoot, assetsRoot
}

func writePlanFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
