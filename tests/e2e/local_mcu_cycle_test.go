//go:build e2e

package e2e

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"debug/elf"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/x12315/rm-relay/internal/execution/command"
)

const (
	profileID = "embedded-stm32f407-robomaster-c"
)

type buildOutputManifest struct {
	SchemaVersion   int                                    `json:"schema_version"`
	ProjectID       string                                 `json:"project_id"`
	ProfileID       string                                 `json:"profile_id"`
	ProfileDigest   string                                 `json:"profile_digest"`
	ProducerVersion string                                 `json:"producer_version"`
	Builder         struct{ ID, Kind string }              `json:"builder"`
	Environment     struct{ ID, Reference, Digest string } `json:"environment"`
	Artifacts       []buildArtifact                        `json:"artifacts"`
}

type buildArtifact struct {
	Role   string `json:"role"`
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

type flashResult struct {
	OK        bool     `json:"ok"`
	Operation string   `json:"operation"`
	ProjectID string   `json:"project_id"`
	Profile   string   `json:"profile"`
	Command   []string `json:"command"`
	Executed  *bool    `json:"executed"`
}

func TestLocalMCUDevelopmentCycle(t *testing.T) {
	environmentReference := os.Getenv("RM_RELAY_E2E_LOCAL_ENVIRONMENT")
	if environmentReference == "" {
		t.Skip("real local E2E requires RM_RELAY_E2E_LOCAL_ENVIRONMENT=image@sha256:digest")
	}
	requireCommand(t, "git")
	requireCommand(t, "docker")

	temporaryRoot := t.TempDir()
	distributionDirectory := filepath.Join(temporaryRoot, "snapshot")
	root := repositoryRoot(t)
	result, err := (command.OSRunner{}).Run(context.Background(), command.Request{Name: "sh", Arguments: []string{"scripts/release/cli.sh", "snapshot", distributionDirectory}, Directory: root})
	if err != nil {
		t.Fatalf("build snapshot: %v: %s", err, result.Stderr)
	}
	archivePath := requireCurrentPlatformArchive(t, distributionDirectory)
	distributedCLI := extractDistributedCLI(t, archivePath, filepath.Join(temporaryRoot, "distribution"))
	producerVersion := assertDistributedVersion(t, distributedCLI)
	projectRoot := cloneProjectTemplate(t, temporaryRoot)
	relayEnvironment := append(
		environmentWithout("RM_RELAY_HOME", "RM_RELAY_MISE_BIN", "RM_RELAY_CACHE_DIR", "RM_RELAY_CONFIG_DIR"),
		"RM_RELAY_CACHE_DIR="+filepath.Join(temporaryRoot, "cache"),
		"RM_RELAY_CONFIG_DIR="+filepath.Join(temporaryRoot, "config"),
	)

	runRelay(t, distributedCLI, relayEnvironment, projectRoot, "environment", "add", "embedded-development", environmentReference, "--builder", "local")
	runRelay(t, distributedCLI, relayEnvironment, projectRoot, "init")
	runRelay(t, distributedCLI, relayEnvironment, projectRoot, "build")
	manifest := assertBuildOutput(t, projectRoot, producerVersion, environmentReference)
	assertFlashDryRun(t, distributedCLI, relayEnvironment, projectRoot, manifest.ProjectID)
}

func cloneProjectTemplate(t *testing.T, temporaryRoot string) string {
	t.Helper()
	templateSource := filepath.Join(repositoryRoot(t), "project-templates", "cross-platform-cpp")
	fixtureRepository := filepath.Join(temporaryRoot, "template-repository")
	copyTree(t, templateSource, fixtureRepository)
	runCommand(t, fixtureRepository, nil, "git", "init", "--quiet")
	runCommand(t, fixtureRepository, nil, "git", "config", "user.name", "RM Relay E2E")
	runCommand(t, fixtureRepository, nil, "git", "config", "user.email", "e2e@rm-relay.invalid")
	runCommand(t, fixtureRepository, nil, "git", "add", ".")
	runCommand(t, fixtureRepository, nil, "git", "commit", "--quiet", "-m", "test fixture")

	projectRoot := filepath.Join(temporaryRoot, "user-project")
	runCommand(t, "", nil, "git", "clone", "--quiet", fixtureRepository, projectRoot)
	return projectRoot
}

func copyTree(t *testing.T, sourceRoot, destinationRoot string) {
	t.Helper()
	err := filepath.WalkDir(sourceRoot, func(sourcePath string, entry fs.DirEntry, walkError error) error {
		if walkError != nil {
			return walkError
		}
		relativePath, err := filepath.Rel(sourceRoot, sourcePath)
		if err != nil {
			return err
		}
		if relativePath == "build" || relativePath == "install" {
			return filepath.SkipDir
		}
		destinationPath := filepath.Join(destinationRoot, relativePath)
		if entry.IsDir() {
			return os.MkdirAll(destinationPath, 0o755)
		}
		if entry.Type()&os.ModeSymlink != 0 || !entry.Type().IsRegular() {
			return fmt.Errorf("template contains unsupported file %q", relativePath)
		}
		source, err := os.Open(sourcePath)
		if err != nil {
			return err
		}
		destination, err := os.OpenFile(destinationPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if err != nil {
			source.Close()
			return err
		}
		_, copyError := io.Copy(destination, source)
		sourceCloseError := source.Close()
		closeError := destination.Close()
		if copyError != nil {
			return copyError
		}
		if sourceCloseError != nil {
			return sourceCloseError
		}
		return closeError
	})
	if err != nil {
		t.Fatalf("copy project template: %v", err)
	}
}

func assertDistributedVersion(t *testing.T, binaryPath string) string {
	t.Helper()
	result := runCommand(t, "", nil, binaryPath, "--version")
	versionOutput := strings.TrimSpace(result.stdout)
	const prefix = "rm-relay version "
	if !strings.HasPrefix(versionOutput, prefix) {
		t.Fatalf("distributed CLI version output = %q", versionOutput)
	}
	version := strings.TrimPrefix(versionOutput, prefix)
	if version == "" || version == "development" {
		t.Fatalf("distributed CLI contains invalid producer version %q", version)
	}
	return version
}

func assertBuildOutput(t *testing.T, projectRoot, producerVersion, environmentReference string) buildOutputManifest {
	t.Helper()
	outputDirectory := filepath.Join(projectRoot, "install", profileID)
	manifestPath := filepath.Join(outputDirectory, "rm-relay-output.json")
	manifestFile, err := os.Open(manifestPath)
	if err != nil {
		t.Fatalf("open Build Output manifest: %v", err)
	}
	defer manifestFile.Close()
	decoder := json.NewDecoder(manifestFile)
	decoder.DisallowUnknownFields()
	var manifest buildOutputManifest
	if err := decoder.Decode(&manifest); err != nil {
		t.Fatalf("decode Build Output manifest: %v", err)
	}
	if manifest.SchemaVersion != 2 || manifest.ProjectID == "" || manifest.ProfileID != profileID {
		t.Fatalf("Build Output identity = %#v", manifest)
	}
	if manifest.ProfileDigest == "" || manifest.Environment.Reference != environmentReference || manifest.Environment.Digest == "" {
		t.Fatalf("Build Output environment identity = %#v", manifest)
	}
	if manifest.ProducerVersion != producerVersion {
		t.Fatalf("Build Output producer version = %q, want %q", manifest.ProducerVersion, producerVersion)
	}
	if len(manifest.Artifacts) != 3 {
		t.Fatalf("Build Output artifacts = %#v, want three files", manifest.Artifacts)
	}
	expectedRoles := []string{"firmware.bin", "firmware.elf", "linker.map"}
	actualRoles := make([]string, 0, len(manifest.Artifacts))
	for _, artifact := range manifest.Artifacts {
		actualRoles = append(actualRoles, artifact.Role)
		assertArtifact(t, outputDirectory, artifact)
		if artifact.Role == "firmware.elf" {
			assertARMELF(t, filepath.Join(outputDirectory, filepath.FromSlash(artifact.Path)))
		}
	}
	slices.Sort(actualRoles)
	if !slices.Equal(actualRoles, expectedRoles) {
		t.Fatalf("Build Output roles = %v, want %v", actualRoles, expectedRoles)
	}
	return manifest
}

func assertArtifact(t *testing.T, outputDirectory string, artifact buildArtifact) {
	t.Helper()
	if !filepath.IsLocal(artifact.Path) || filepath.Clean(artifact.Path) == "." || strings.Contains(artifact.Path, `\`) {
		t.Fatalf("artifact %q has unsafe path %q", artifact.Role, artifact.Path)
	}
	artifactPath := filepath.Join(outputDirectory, filepath.FromSlash(artifact.Path))
	file, err := os.Open(artifactPath)
	if err != nil {
		t.Fatalf("open artifact %q: %v", artifact.Role, err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		t.Fatalf("inspect artifact %q: %v", artifact.Role, err)
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		t.Fatalf("hash artifact %q: %v", artifact.Role, err)
	}
	actualHash := hex.EncodeToString(hash.Sum(nil))
	if info.Size() != artifact.Size || actualHash != artifact.SHA256 {
		t.Fatalf("artifact %q identity changed: size=%d hash=%s", artifact.Role, info.Size(), actualHash)
	}
}

func assertARMELF(t *testing.T, path string) {
	t.Helper()
	firmware, err := elf.Open(path)
	if err != nil {
		t.Fatalf("open firmware ELF: %v", err)
	}
	defer firmware.Close()
	if firmware.Class != elf.ELFCLASS32 || firmware.Data != elf.ELFDATA2LSB || firmware.Machine != elf.EM_ARM {
		t.Fatalf("firmware ELF class/data/machine = %s/%s/%s, want ELF32/LSB/ARM", firmware.Class, firmware.Data, firmware.Machine)
	}
}

func assertFlashDryRun(t *testing.T, binaryPath string, environment []string, projectRoot, projectID string) {
	t.Helper()
	result := runRelay(t, binaryPath, environment, projectRoot, "--format", "json", "flash", "--target", "openocd-stlink", "--dry-run")
	decoder := json.NewDecoder(strings.NewReader(result.stdout))
	decoder.DisallowUnknownFields()
	var flash flashResult
	if err := decoder.Decode(&flash); err != nil {
		t.Fatalf("decode flash result: %v\n%s", err, result.stdout)
	}
	if !flash.OK || flash.Operation != "flash" || flash.ProjectID != projectID || flash.Profile != profileID {
		t.Fatalf("flash result identity = %#v", flash)
	}
	if flash.Executed == nil || *flash.Executed {
		t.Fatalf("flash dry-run execution state = %#v", flash.Executed)
	}
	if len(flash.Command) == 0 || !slices.Contains(flash.Command, "openocd") {
		t.Fatalf("flash dry-run command = %#v", flash.Command)
	}
}

func runRelay(t *testing.T, binaryPath string, environment []string, projectRoot string, arguments ...string) commandResult {
	t.Helper()
	commandArguments := append([]string{"--project", projectRoot}, arguments...)
	return runCommand(t, "", environment, binaryPath, commandArguments...)
}

type commandResult struct {
	stdout string
	stderr string
}

func runCommand(t *testing.T, directory string, environment []string, name string, arguments ...string) commandResult {
	t.Helper()
	command := exec.Command(name, arguments...)
	command.Dir = directory
	if environment != nil {
		command.Env = environment
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		t.Fatalf("run %s %s: %v\nstdout:\n%s\nstderr:\n%s", name, strings.Join(arguments, " "), err, stdout.String(), stderr.String())
	}
	return commandResult{stdout: stdout.String(), stderr: stderr.String()}
}

func environmentWithout(names ...string) []string {
	excluded := make(map[string]struct{}, len(names))
	for _, name := range names {
		excluded[name] = struct{}{}
	}
	environment := make([]string, 0, len(os.Environ()))
	for _, entry := range os.Environ() {
		name, _, _ := strings.Cut(entry, "=")
		if _, exists := excluded[name]; !exists {
			environment = append(environment, entry)
		}
	}
	return environment
}

func extractDistributedCLI(t *testing.T, archivePath, destinationDirectory string) string {
	t.Helper()
	if err := os.MkdirAll(destinationDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	binaryName := "rm-relay"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	destinationPath := filepath.Join(destinationDirectory, binaryName)
	if strings.HasSuffix(archivePath, ".zip") {
		extractCLIFromZip(t, archivePath, binaryName, destinationPath)
	} else {
		extractCLIFromTarGzip(t, archivePath, binaryName, destinationPath)
	}
	if err := os.Chmod(destinationPath, 0o755); err != nil {
		t.Fatalf("make distributed CLI executable: %v", err)
	}
	return destinationPath
}

func extractCLIFromZip(t *testing.T, archivePath, binaryName, destinationPath string) {
	t.Helper()
	archive, err := zip.OpenReader(archivePath)
	if err != nil {
		t.Fatalf("open CLI archive: %v", err)
	}
	defer archive.Close()
	for _, file := range archive.File {
		if file.Name != binaryName {
			continue
		}
		source, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		writeExtractedCLI(t, source, destinationPath)
		source.Close()
		return
	}
	t.Fatalf("CLI archive %s does not contain %s", archivePath, binaryName)
}

func extractCLIFromTarGzip(t *testing.T, archivePath, binaryName, destinationPath string) {
	t.Helper()
	archive, err := os.Open(archivePath)
	if err != nil {
		t.Fatalf("open CLI archive: %v", err)
	}
	defer archive.Close()
	gzipReader, err := gzip.NewReader(archive)
	if err != nil {
		t.Fatalf("open CLI gzip stream: %v", err)
	}
	defer gzipReader.Close()
	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("read CLI archive: %v", err)
		}
		if header.Name == binaryName {
			writeExtractedCLI(t, tarReader, destinationPath)
			return
		}
	}
	t.Fatalf("CLI archive %s does not contain %s", archivePath, binaryName)
}

func writeExtractedCLI(t *testing.T, source io.Reader, destinationPath string) {
	t.Helper()
	destination, err := os.OpenFile(destinationPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o755)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(destination, source); err != nil {
		destination.Close()
		t.Fatal(err)
	}
	if err := destination.Close(); err != nil {
		t.Fatal(err)
	}
}

func requireCommand(t *testing.T, name string) string {
	t.Helper()
	path, err := exec.LookPath(name)
	if err != nil {
		t.Fatalf("required command %q was not found in PATH; install it before running mise run test:e2e", name)
	}
	return path
}

func requireCurrentPlatformArchive(t *testing.T, distributionDirectory string) string {
	t.Helper()
	extension := ".tar.gz"
	if runtime.GOOS == "windows" {
		extension = ".zip"
	}
	pattern := filepath.Join(distributionDirectory, fmt.Sprintf("rm-relay_*_%s_%s%s", runtime.GOOS, runtime.GOARCH, extension))
	matches, err := filepath.Glob(pattern)
	if err != nil {
		t.Fatalf("match current-platform CLI archive: %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("current-platform CLI archive pattern %q matched %d files", pattern, len(matches))
	}
	return matches[0]
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, sourcePath, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate E2E test source")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(sourcePath), "..", ".."))
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("locate repository root: %v", err)
	}
	return root
}
