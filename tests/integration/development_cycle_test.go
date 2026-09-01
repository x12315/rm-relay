package integration

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/x12315/rm-relay/internal/build/output"
)

func TestDevelopmentCycleComposesProjectBuildOutputAndFlashTarget(t *testing.T) {
	fixture := newDevelopmentCycleFixture(t)

	initResult := fixture.runCLI(t, "init")
	if initResult.exitCode != 0 {
		t.Fatalf("init exit code = %d, stderr = %s", initResult.exitCode, initResult.stderr)
	}
	buildResult := fixture.runCLI(t, "build")
	if buildResult.exitCode != 0 {
		t.Fatalf("build exit code = %d, stderr = %s", buildResult.exitCode, buildResult.stderr)
	}
	flashResult := fixture.runCLI(t, "--format", "json", "flash", "--target", "openocd-stlink", "--dry-run")
	if flashResult.exitCode != 0 {
		t.Fatalf("flash exit code = %d, stderr = %s", flashResult.exitCode, flashResult.stderr)
	}

	manifestPath := filepath.Join(fixture.outputDirectory(), output.ManifestFileName)
	manifest := readBuildOutputManifest(t, manifestPath)
	if manifest.ProjectID == "" || manifest.ProfileID != testProfileID || manifest.ProducerVersion != testProducerVersion {
		t.Fatalf("Build Output identity = %#v", manifest)
	}
	if len(fixture.builders.prepared) != 1 || fixture.builders.prepared[0] != "local" {
		t.Fatalf("prepared Builders = %v", fixture.builders.prepared)
	}

	var result struct {
		OK        bool     `json:"ok"`
		Operation string   `json:"operation"`
		Executed  bool     `json:"executed"`
		Command   []string `json:"command"`
	}
	if err := json.Unmarshal([]byte(flashResult.stdout), &result); err != nil {
		t.Fatalf("decode flash result: %v\n%s", err, flashResult.stdout)
	}
	if !result.OK || result.Operation != "flash" || result.Executed || !contains(result.Command, "openocd") {
		t.Fatalf("flash result = %#v", result)
	}
	if got := len(fixture.runner.requests); got != 0 {
		t.Fatalf("process count = %d, want no host process for dry-run integration", got)
	}
}
