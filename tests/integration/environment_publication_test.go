package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestEnvironmentPublicationUsesOneBakeContractAndWritesExternalHandoff(t *testing.T) {
	repositoryRoot := integrationRepositoryRoot(t)
	temporaryRoot := t.TempDir()
	binDirectory := filepath.Join(temporaryRoot, "bin")
	if err := os.Mkdir(binDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	dockerLog := filepath.Join(temporaryRoot, "docker.log")
	manifestPath := filepath.Join(temporaryRoot, "manifest.json")
	manifest := `{
  "schemaVersion": 2,
  "mediaType": "application/vnd.oci.image.index.v1+json",
  "digest": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
  "manifests": [
    {"platform":{"architecture":"amd64","os":"linux"}},
    {"platform":{"architecture":"arm64","os":"linux"}}
  ]
}`
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	fakeDocker := `#!/bin/sh
set -eu
printf '%s\n' "$*" >> "$DOCKER_LOG"
case "$*" in
  *"imagetools inspect"*) cat "$DOCKER_MANIFEST" ;;
esac
`
	dockerPath := filepath.Join(binDirectory, "docker")
	if err := os.WriteFile(dockerPath, []byte(fakeDocker), 0o700); err != nil {
		t.Fatal(err)
	}
	fakeGit := `#!/bin/sh
set -eu
case "$*" in
  *"status --porcelain"*) exit 0 ;;
  *"rev-parse HEAD"*) printf '%s\n' 'bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb' ;;
  *) exit 1 ;;
esac
`
	if err := os.WriteFile(filepath.Join(binDirectory, "git"), []byte(fakeGit), 0o700); err != nil {
		t.Fatal(err)
	}
	handoffPath := filepath.Join(temporaryRoot, "embedded-development.toml")
	command := exec.Command("sh", filepath.Join(repositoryRoot, "environments", "embedded-development", "publish.sh"), "image-factory", "registry.example.org/rm-relay/embedded-development:v0.1.0", handoffPath)
	command.Dir = repositoryRoot
	command.Env = append(os.Environ(), "PATH="+binDirectory+string(os.PathListSeparator)+os.Getenv("PATH"), "DOCKER_LOG="+dockerLog, "DOCKER_MANIFEST="+manifestPath)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("publish.sh: %v\n%s", err, output)
	}

	log, err := os.ReadFile(dockerLog)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		"buildx bake --builder image-factory",
		"--check publish",
		"mcu-dev-multiarch.tags=registry.example.org/rm-relay/embedded-development:v0.1.0",
		"--push",
		"--provenance mode=max",
		"--sbom true",
		"buildx imagetools inspect registry.example.org/rm-relay/embedded-development:v0.1.0 --format {{json .Manifest}}",
	} {
		if !strings.Contains(string(log), expected) {
			t.Fatalf("Docker log does not contain %q:\n%s", expected, log)
		}
	}
	handoff, err := os.ReadFile(handoffPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		`environment_id = "embedded-development"`,
		`tag = "registry.example.org/rm-relay/embedded-development:v0.1.0"`,
		`digest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"`,
		`immutable_reference = "registry.example.org/rm-relay/embedded-development@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"`,
		`source_revision = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"`,
		`platforms = ["linux/amd64", "linux/arm64"]`,
	} {
		if !strings.Contains(string(handoff), expected) {
			t.Fatalf("handoff does not contain %q:\n%s", expected, handoff)
		}
	}
}

func TestEnvironmentPublicationRejectsRepositoryOutput(t *testing.T) {
	repositoryRoot := integrationRepositoryRoot(t)
	outputPath := filepath.Join(repositoryRoot, "environment-publication.toml")
	command := exec.Command("sh", filepath.Join(repositoryRoot, "environments", "embedded-development", "publish.sh"), "image-factory", "registry.example.org/rm-relay/embedded-development:v0.1.0", outputPath)
	command.Dir = repositoryRoot
	output, err := command.CombinedOutput()
	if err == nil || !strings.Contains(string(output), "outside the repository") {
		t.Fatalf("publish.sh error = %v, output = %s", err, output)
	}
	if _, err := os.Lstat(outputPath); !os.IsNotExist(err) {
		t.Fatalf("publication created repository output: %v", err)
	}
}

func integrationRepositoryRoot(t *testing.T) string {
	t.Helper()
	_, sourcePath, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate integration test source")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(sourcePath), "..", ".."))
}
