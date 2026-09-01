package environmentimagebuilder_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

const (
	independentBakeFile     = "docker-bake.hcl"
	independentIdentityFile = "identity.toml"
)

func TestPublishUsesExplicitEnvironmentSourceAndWritesExternalHandoff(t *testing.T) {
	repositoryRoot := serviceRepositoryRoot(t)
	temporaryRoot := t.TempDir()
	environmentSource := createEnvironmentSource(t, temporaryRoot)
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
	writeExecutable(t, filepath.Join(binDirectory, "docker"), `#!/bin/sh
set -eu
printf '%s\n' "$*" >> "$DOCKER_LOG"
case "$*" in
  *"imagetools inspect"*) cat "$DOCKER_MANIFEST" ;;
  *"buildx bake"*"--metadata-file"*)
    while [ "$#" -gt 0 ]; do
      if [ "$1" = "--metadata-file" ]; then
        shift
        printf '%s\n' '{"containerimage.digest":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}' > "$1"
        break
      fi
      shift
    done
    ;;
esac
`)
	writeExecutable(t, filepath.Join(binDirectory, "git"), `#!/bin/sh
set -eu
case "$*" in
  *"status --porcelain"*) exit 0 ;;
  *"rev-parse --show-toplevel"*) printf '%s\n' "$SOURCE_ROOT" ;;
  *"rev-parse HEAD"*) printf '%s\n' 'bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb' ;;
  *) exit 1 ;;
esac
`)
	handoffPath := filepath.Join(temporaryRoot, "embedded-development.toml")
	command := publishCommand(repositoryRoot, environmentSource, handoffPath)
	command.Env = append(os.Environ(),
		"PATH="+binDirectory+string(os.PathListSeparator)+os.Getenv("PATH"),
		"DOCKER_LOG="+dockerLog,
		"DOCKER_MANIFEST="+manifestPath,
		"SOURCE_ROOT="+environmentSource,
	)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("publish.sh: %v\n%s", err, output)
	}

	log, err := os.ReadFile(dockerLog)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		"buildx bake --builder image-factory --file " + independentBakeFile + " --check publish",
		"buildx bake --builder image-factory --file " + independentBakeFile + " publish",
		"--set *.tags=registry.example.org/rm-relay/embedded-development:v0.1.0",
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

func TestPublishRejectsOutputInsideEnvironmentSource(t *testing.T) {
	repositoryRoot := serviceRepositoryRoot(t)
	environmentSource := createEnvironmentSource(t, t.TempDir())
	outputPath := filepath.Join(environmentSource, "environment-publication.toml")
	command := publishCommand(repositoryRoot, environmentSource, outputPath)
	output, err := command.CombinedOutput()
	if err == nil || !strings.Contains(string(output), "outside the environment source") {
		t.Fatalf("publish.sh error = %v, output = %s", err, output)
	}
	if _, err := os.Lstat(outputPath); !os.IsNotExist(err) {
		t.Fatalf("publication created source output: %v", err)
	}
}

func TestPublishRejectsDefinitionFilesOutsideEnvironmentSource(t *testing.T) {
	repositoryRoot := serviceRepositoryRoot(t)
	environmentSource := createEnvironmentSource(t, t.TempDir())
	command := exec.Command("sh", filepath.Join(repositoryRoot, "services", "environment-image-builder", "publish.sh"),
		environmentSource,
		"../docker-bake.hcl",
		independentIdentityFile,
		"image-factory",
		"registry.example.org/rm-relay/embedded-development:v0.1.0",
		filepath.Join(t.TempDir(), "handoff.toml"),
	)
	command.Dir = repositoryRoot
	output, err := command.CombinedOutput()
	if err == nil || !strings.Contains(string(output), "Bake file must stay inside the environment source") {
		t.Fatalf("publish.sh error = %v, output = %s", err, output)
	}
}

func TestPublishRejectsOutputWhoseParentResolvesInsideEnvironmentSource(t *testing.T) {
	repositoryRoot := serviceRepositoryRoot(t)
	temporaryRoot := t.TempDir()
	environmentSource := createEnvironmentSource(t, temporaryRoot)
	linkedParent := filepath.Join(temporaryRoot, "linked-output")
	if err := os.Symlink(environmentSource, linkedParent); err != nil {
		t.Skipf("create output symlink: %v", err)
	}
	command := publishCommand(repositoryRoot, environmentSource, filepath.Join(linkedParent, "handoff.toml"))
	output, err := command.CombinedOutput()
	if err == nil || !strings.Contains(string(output), "outside the environment source") {
		t.Fatalf("publish.sh error = %v, output = %s", err, output)
	}
}

func publishCommand(repositoryRoot, environmentSource, handoffPath string) *exec.Cmd {
	command := exec.Command("sh", filepath.Join(repositoryRoot, "services", "environment-image-builder", "publish.sh"),
		environmentSource,
		independentBakeFile,
		independentIdentityFile,
		"image-factory",
		"registry.example.org/rm-relay/embedded-development:v0.1.0",
		handoffPath,
	)
	command.Dir = repositoryRoot
	return command
}

func createEnvironmentSource(t *testing.T, parent string) string {
	t.Helper()
	root := filepath.Join(parent, "environment-source")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, independentBakeFile), []byte("group \"publish\" {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, independentIdentityFile), []byte("schema_version = 1\nid = \"embedded-development\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return root
}

func writeExecutable(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o700); err != nil {
		t.Fatal(err)
	}
}

func serviceRepositoryRoot(t *testing.T) string {
	t.Helper()
	_, sourcePath, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate environment image builder test source")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(sourcePath), "..", ".."))
}
