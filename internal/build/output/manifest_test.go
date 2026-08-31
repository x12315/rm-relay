package output_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/x12315/rm-relay/internal/build"
	"github.com/x12315/rm-relay/internal/build/output"
	"github.com/x12315/rm-relay/internal/profile"
	"github.com/x12315/rm-relay/internal/project"
)

func TestCreateWritesDeterministicManifestWithoutAbsolutePathsOrTime(t *testing.T) {
	plan := buildOutputPlan(t)
	writeArtifacts(t, plan.OutputDirectory)

	if _, err := output.Create(createRequest(plan)); err != nil {
		t.Fatalf("first output.Create() error = %v", err)
	}
	manifestPath := filepath.Join(plan.OutputDirectory, output.ManifestFileName)
	first, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := output.Create(createRequest(plan)); err != nil {
		t.Fatalf("second output.Create() error = %v", err)
	}
	second, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Fatal("manifest bytes changed for identical inputs")
	}
	if strings.Contains(string(first), plan.ProjectRoot) || strings.Contains(string(first), "timestamp") || strings.Contains(string(first), "created_at") {
		t.Fatalf("manifest contains an absolute path or time field:\n%s", first)
	}
	if !strings.Contains(string(first), `"schema_version": 2`) || !strings.Contains(string(first), `"builder"`) || !strings.Contains(string(first), `"environment"`) {
		t.Fatalf("manifest does not use schema v2 evidence:\n%s", first)
	}
}

func TestCreateRejectsMalformedEnvironmentDigest(t *testing.T) {
	plan := buildOutputPlan(t)
	writeArtifacts(t, plan.OutputDirectory)
	request := createRequest(plan)
	request.Environment.Digest = "sha256:short"
	if _, err := output.Create(request); err == nil {
		t.Fatal("malformed environment digest accepted")
	}
}

func TestCreateRejectsMissingRequiredArtifact(t *testing.T) {
	plan := buildOutputPlan(t)
	writeOutputFile(t, plan.OutputDirectory, "firmware.elf", "elf")

	_, err := output.Create(createRequest(plan))
	if err == nil || !strings.Contains(err.Error(), "firmware.bin") {
		t.Fatalf("output.Create() error = %v, want missing firmware.bin", err)
	}
}

func TestLoadAndValidateRejectsProfileDigestMismatch(t *testing.T) {
	plan := buildOutputPlan(t)
	writeArtifacts(t, plan.OutputDirectory)
	if _, err := output.Create(createRequest(plan)); err != nil {
		t.Fatal(err)
	}
	changedProfile := plan.Profile
	changedProfile.Digest = strings.Repeat("f", 64)

	_, err := output.LoadAndValidate(plan.OutputDirectory, plan.ProjectID, changedProfile)
	if err == nil || !strings.Contains(err.Error(), "profile digest") {
		t.Fatalf("output.LoadAndValidate() error = %v, want profile digest mismatch", err)
	}
}

func TestLoadAndValidateRejectsArtifactHashMismatch(t *testing.T) {
	plan := buildOutputPlan(t)
	writeArtifacts(t, plan.OutputDirectory)
	if _, err := output.Create(createRequest(plan)); err != nil {
		t.Fatal(err)
	}
	writeOutputFile(t, plan.OutputDirectory, "firmware.elf", "ELF")

	_, err := output.LoadAndValidate(plan.OutputDirectory, plan.ProjectID, plan.Profile)
	if err == nil || !strings.Contains(err.Error(), "SHA-256") {
		t.Fatalf("output.LoadAndValidate() error = %v, want hash mismatch", err)
	}
}

func TestLoadAndValidateRejectsArtifactPathOutsideOutputRoot(t *testing.T) {
	plan := buildOutputPlan(t)
	writeArtifacts(t, plan.OutputDirectory)
	if _, err := output.Create(createRequest(plan)); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(plan.OutputDirectory, output.ManifestFileName)
	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	modified := strings.Replace(string(manifestBytes), `"path": "firmware.elf"`, `"path": "../firmware.elf"`, 1)
	if err := os.WriteFile(manifestPath, []byte(modified), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err = output.LoadAndValidate(plan.OutputDirectory, plan.ProjectID, plan.Profile)
	if err == nil || !strings.Contains(err.Error(), "safe relative") {
		t.Fatalf("output.LoadAndValidate() error = %v, want path boundary error", err)
	}
}

func TestLoadAndValidateReturnsVerifiedArtifactPath(t *testing.T) {
	plan := buildOutputPlan(t)
	writeArtifacts(t, plan.OutputDirectory)
	if _, err := output.Create(createRequest(plan)); err != nil {
		t.Fatal(err)
	}

	verified, err := output.LoadAndValidate(plan.OutputDirectory, plan.ProjectID, plan.Profile)
	if err != nil {
		t.Fatalf("output.LoadAndValidate() error = %v", err)
	}
	artifactPath, err := verified.ArtifactPathByRole("firmware.elf")
	if err != nil {
		t.Fatalf("ArtifactPathByRole() error = %v", err)
	}
	want := filepath.Join(plan.OutputDirectory, "firmware.elf")
	if artifactPath != want {
		t.Fatalf("ArtifactPathByRole() = %q, want %q", artifactPath, want)
	}
}

func buildOutputPlan(t *testing.T) build.Plan {
	t.Helper()
	projectRoot := t.TempDir()
	return build.Plan{
		Operation:       build.OperationBuild,
		ProjectRoot:     projectRoot,
		ProjectID:       "1e013e16-04a7-4fd3-9f48-bfc9178f5421",
		OutputDirectory: filepath.Join(projectRoot, "install", "embedded-test"),
		Build: project.Build{
			Profile: "embedded-test",
			Outputs: []project.Output{
				{Role: "firmware.elf", Path: "firmware.elf"},
				{Role: "firmware.bin", Path: "firmware.bin"},
			},
		},
		Profile: profile.Loaded{
			Digest: strings.Repeat("a", 64),
			Config: profile.Config{
				ID:                  "embedded-test",
				Environment:         profile.Environment{ID: "embedded-development", LocalReference: "mcu-dev/toolchain:test"},
				RequiredOutputRoles: []string{"firmware.elf", "firmware.bin"},
			},
		},
	}
}

func createRequest(plan build.Plan) output.CreateRequest {
	return output.CreateRequest{
		OutputDirectory: plan.OutputDirectory,
		ProjectID:       plan.ProjectID,
		Profile:         plan.Profile,
		DeclaredOutputs: plan.Build.Outputs,
		Builder:         output.BuilderEvidence{ID: "local", Kind: "local-container"},
		Environment:     output.EnvironmentEvidence{ID: "embedded-development", Reference: "mcu-dev/toolchain:test", Digest: "sha256:" + strings.Repeat("b", 64)},
		ProducerVersion: "0.1.0",
	}
}

func writeArtifacts(t *testing.T, outputDirectory string) {
	t.Helper()
	writeOutputFile(t, outputDirectory, "firmware.elf", "elf")
	writeOutputFile(t, outputDirectory, "firmware.bin", "bin")
}

func writeOutputFile(t *testing.T, outputDirectory, name, content string) {
	t.Helper()
	if err := os.MkdirAll(outputDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outputDirectory, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
