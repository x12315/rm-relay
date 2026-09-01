//go:build e2e

package e2e

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRemoteMCUDevelopmentCycle(t *testing.T) {
	required := map[string]string{
		"endpoint": "RM_RELAY_E2E_REMOTE_ENDPOINT", "ca": "RM_RELAY_E2E_REMOTE_CA",
		"certificate": "RM_RELAY_E2E_REMOTE_CERT", "key": "RM_RELAY_E2E_REMOTE_KEY",
		"server name": "RM_RELAY_E2E_REMOTE_SERVER_NAME", "environment": "RM_RELAY_E2E_REMOTE_ENVIRONMENT",
	}
	values := make(map[string]string, len(required))
	for label, variable := range required {
		values[label] = os.Getenv(variable)
		if values[label] == "" {
			t.Skipf("real remote E2E requires %s (%s)", label, variable)
		}
	}
	requireCommand(t, "git")
	requireCommand(t, "docker")
	temporaryRoot := t.TempDir()
	distributedCLI := buildCurrentPlatformCLI(t, temporaryRoot)
	projectRoot := cloneProjectTemplate(t, temporaryRoot)
	environment := append(environmentWithout("RM_RELAY_CONFIG_DIR", "RM_RELAY_CACHE_DIR"), "RM_RELAY_CONFIG_DIR="+filepath.Join(temporaryRoot, "config"), "RM_RELAY_CACHE_DIR="+filepath.Join(temporaryRoot, "cache"))
	runRelay(t, distributedCLI, environment, projectRoot, "builder", "add", "team", "--endpoint", values["endpoint"], "--ca", values["ca"], "--cert", values["certificate"], "--key", values["key"], "--server-name", values["server name"])
	runRelay(t, distributedCLI, environment, projectRoot, "environment", "add", "embedded-development", values["environment"], "--builder", "team")
	runRelay(t, distributedCLI, environment, projectRoot, "builder", "check", "team")
	runRelay(t, distributedCLI, environment, projectRoot, "init")
	runRelay(t, distributedCLI, environment, projectRoot, "build", "--builder", "team")
	producerVersion := assertDistributedVersion(t, distributedCLI)
	manifest := assertBuildOutput(t, projectRoot, producerVersion, values["environment"])
	if manifest.Builder.ID != "team" || manifest.Builder.Kind != "remote-buildkit" {
		t.Fatalf("remote Builder evidence = %#v", manifest.Builder)
	}
}

func buildCurrentPlatformCLI(t *testing.T, temporaryRoot string) string {
	t.Helper()
	distributionDirectory := filepath.Join(temporaryRoot, "snapshot")
	result := runCommand(t, repositoryRoot(t), environmentWithout("RM_RELAY_CLI_OUTPUT_DIR"), "sh", "scripts/release/cli.sh", "snapshot", distributionDirectory)
	_ = result
	archivePath := requireCurrentPlatformArchive(t, distributionDirectory)
	return extractDistributedCLI(t, archivePath, filepath.Join(temporaryRoot, "distribution"))
}
