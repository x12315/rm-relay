package project

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestLoadRejectsUnknownSchemaVersion(t *testing.T) {
	projectRoot := writeProjectConfig(t, `schema_version = 2
project_id = "1e013e16-04a7-4fd3-9f48-bfc9178f5421"
default_profile = "embedded-test"
`)

	_, err := Load(projectRoot)
	if err == nil || !strings.Contains(err.Error(), "schema_version") {
		t.Fatalf("Load() error = %v, want unsupported schema_version", err)
	}
}

func TestLoadRejectsOutputPathOutsideInstallRoot(t *testing.T) {
	projectRoot := writeProjectConfig(t, validProjectConfig("../firmware.elf"))

	_, err := Load(projectRoot)
	if err == nil || !strings.Contains(err.Error(), "output path") {
		t.Fatalf("Load() error = %v, want invalid output path", err)
	}
}

func TestBuildForProfileReturnsExactlyOneDeclaration(t *testing.T) {
	config := Config{Builds: []Build{
		{Profile: "embedded-test", System: "cmake", Preset: "first"},
		{Profile: "embedded-test", System: "cmake", Preset: "second"},
	}}

	_, err := config.BuildForProfile("embedded-test")
	if err == nil || !strings.Contains(err.Error(), "multiple") {
		t.Fatalf("BuildForProfile() error = %v, want duplicate declaration error", err)
	}
}

func TestLoadRejectsProjectOwnedMiseExecutionDetails(t *testing.T) {
	projectRoot := writeProjectConfig(t, `schema_version = 1
project_id = ""
default_profile = "embedded-test"

[[builds]]
profile = "embedded-test"
system = "cmake"
preset = "embedded-test"
mise_config = "mise.toml"
task = "build:firmware"

[[builds.outputs]]
role = "firmware.elf"
path = "firmware.elf"
`)

	_, err := Load(projectRoot)
	if err == nil || !strings.Contains(err.Error(), "unknown key") {
		t.Fatalf("Load() error = %v, want project-owned mise fields rejected", err)
	}
}

func TestInitializeReplacesOnlyBlankProjectIdentity(t *testing.T) {
	const original = `# This comment must survive initialization.
schema_version = 1
project_id = ""
default_profile = "embedded-test"

[[builds]]
profile = "embedded-test"
system = "cmake"
preset = "embedded-test"

[[builds.outputs]]
role = "firmware.elf"
path = "firmware.elf"
`
	projectRoot := writeProjectConfig(t, original)

	projectID, err := Initialize(projectRoot)
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	if !regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`).MatchString(projectID) {
		t.Fatalf("Initialize() projectID = %q, want UUID v4", projectID)
	}

	updatedBytes, err := os.ReadFile(filepath.Join(projectRoot, FileName))
	if err != nil {
		t.Fatal(err)
	}
	expected := strings.Replace(original, `project_id = ""`, `project_id = "`+projectID+`"`, 1)
	if string(updatedBytes) != expected {
		t.Fatalf("initialized config changed unrelated content\nwant:\n%s\ngot:\n%s", expected, updatedBytes)
	}
}

func TestInitializePreservesExistingProjectIdentity(t *testing.T) {
	const projectID = "1e013e16-04a7-4fd3-9f48-bfc9178f5421"
	projectRoot := writeProjectConfig(t, strings.Replace(
		validProjectConfig("firmware.elf"),
		`project_id = ""`,
		`project_id = "`+projectID+`"`,
		1,
	))
	configPath := filepath.Join(projectRoot, FileName)
	before, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}

	actualID, err := Initialize(projectRoot)
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	after, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if actualID != projectID {
		t.Fatalf("Initialize() projectID = %q, want %q", actualID, projectID)
	}
	if string(after) != string(before) {
		t.Fatal("Initialize() rewrote a project that already had an identity")
	}
}

func writeProjectConfig(t *testing.T, content string) string {
	t.Helper()
	projectRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectRoot, FileName), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return projectRoot
}

func validProjectConfig(outputPath string) string {
	return `schema_version = 1
project_id = ""
default_profile = "embedded-test"

[[builds]]
profile = "embedded-test"
system = "cmake"
preset = "embedded-test"

[[builds.outputs]]
role = "firmware.elf"
path = "` + outputPath + `"
`
}
