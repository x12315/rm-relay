package build

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/x12315/rm-relay/internal/profile"
)

func TestResolveUsesProjectDefaultProfile(t *testing.T) {
	projectRoot, profiles := writePlanFixture(t)

	plan, err := Resolve(OperationBuild, projectRoot, "", "", profiles)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if plan.Profile.Config.ID != "embedded-default" {
		t.Fatalf("Profile.ID = %q", plan.Profile.Config.ID)
	}
	if plan.Build.System != "cmake" || plan.Build.Preset != "default-preset" {
		t.Fatalf("Build = %+v", plan.Build)
	}
	wantOutput := filepath.Join(projectRoot, "install", "embedded-default")
	if plan.OutputDirectory != wantOutput {
		t.Fatalf("OutputDirectory = %q, want %q", plan.OutputDirectory, wantOutput)
	}
}

func TestResolveUsesExplicitProfileOverride(t *testing.T) {
	projectRoot, profiles := writePlanFixture(t)

	plan, err := Resolve(OperationFlash, projectRoot, "embedded-override", "", profiles)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if plan.Operation != OperationFlash {
		t.Fatalf("Operation = %q", plan.Operation)
	}
	if plan.Profile.Config.ID != "embedded-override" || plan.Build.Preset != "override-preset" {
		t.Fatalf("resolved override = profile %q, preset %q", plan.Profile.Config.ID, plan.Build.Preset)
	}
}

func writePlanFixture(t *testing.T) (string, profile.Catalog) {
	t.Helper()
	projectRoot := t.TempDir()
	projectConfig := `schema_version = 1
project_id = "1e013e16-04a7-4fd3-9f48-bfc9178f5421"
default_profile = "embedded-default"

[[builds]]
profile = "embedded-default"
system = "cmake"
preset = "default-preset"

[[builds.outputs]]
role = "firmware.elf"
path = "default.elf"

[[builds]]
profile = "embedded-override"
system = "cmake"
preset = "override-preset"

[[builds.outputs]]
role = "firmware.elf"
path = "override.elf"
`
	if err := os.WriteFile(filepath.Join(projectRoot, "rm-relay.toml"), []byte(projectConfig), 0o644); err != nil {
		t.Fatal(err)
	}
	files := fstest.MapFS{}
	for _, profileID := range []string{"embedded-default", "embedded-override"} {
		files["profiles/"+profileID+"/profile.toml"] = &fstest.MapFile{Data: []byte(`schema_version = 2
id = "` + profileID + `"
required_output_roles = ["firmware.elf"]

[environment]
id = "embedded-development"
`)}
	}
	return projectRoot, profile.Catalog{Files: files, Root: "profiles"}
}
