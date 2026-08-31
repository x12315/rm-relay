// Package distribution builds local CLI candidates without writing generated files into the source repository.
package packager

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/x12315/rm-relay/internal/execution/command"
)

const goReleaserConfig = "distribution/cli/goreleaser.yaml"

// Binary identifies one host CLI produced by GoReleaser.
type Binary struct {
	Path    string
	Version string
	SHA256  string
}

// Packager runs the repository GoReleaser configuration in an expendable external clone.
type Packager struct {
	Runner         command.Runner
	RepositoryRoot string
	GoReleaser     string
	Progress       io.Writer
}

// CrossBuild creates the unarchived cross-platform CLI matrix in a new external directory.
func (packager Packager) CrossBuild(ctx context.Context, outputDirectory string) error {
	return packager.publishDist(ctx, outputDirectory, []string{"build", "--snapshot", "--clean"})
}

// PackageSnapshot creates the cross-platform CLI snapshot archives in a new external directory.
func (packager Packager) PackageSnapshot(ctx context.Context, outputDirectory string) error {
	return packager.publishDist(ctx, outputDirectory, []string{"release", "--snapshot", "--clean"})
}

// BuildHostBinary creates one current-platform CLI and returns its version and content identity.
func (packager Packager) BuildHostBinary(ctx context.Context, outputPath string) (Binary, error) {
	repositoryRoot, destinationPath, err := packager.validatedDestination(outputPath)
	if err != nil {
		return Binary{}, err
	}
	if err := requireAbsent(destinationPath); err != nil {
		return Binary{}, err
	}
	if err := packager.requireCleanRepository(ctx, repositoryRoot); err != nil {
		return Binary{}, err
	}
	if err := os.MkdirAll(filepath.Dir(destinationPath), 0o755); err != nil {
		return Binary{}, fmt.Errorf("create CLI output parent: %w", err)
	}

	checkout, cleanup, err := packager.cloneRepository(ctx, repositoryRoot)
	if err != nil {
		return Binary{}, err
	}
	defer cleanup()

	result, err := packager.Runner.Run(ctx, command.Request{
		Name:      packager.goReleaserBinary(),
		Arguments: []string{"build", "--snapshot", "--clean", "--config", goReleaserConfig, "--single-target", "--output", destinationPath},
		Directory: checkout,
		Stdout:    packager.Progress,
		Stderr:    packager.Progress,
	})
	if err != nil {
		return Binary{}, processFailure("build host CLI", result, err)
	}
	versionResult, err := packager.Runner.Run(ctx, command.Request{Name: destinationPath, Arguments: []string{"--version"}})
	if err != nil {
		return Binary{}, processFailure("read host CLI version", versionResult, err)
	}
	version, err := parseVersion(versionResult.Stdout)
	if err != nil {
		return Binary{}, err
	}
	digest, err := fileSHA256(destinationPath)
	if err != nil {
		return Binary{}, err
	}
	return Binary{Path: destinationPath, Version: version, SHA256: digest}, nil
}

func (packager Packager) publishDist(ctx context.Context, outputDirectory string, arguments []string) error {
	repositoryRoot, destinationDirectory, err := packager.validatedDestination(outputDirectory)
	if err != nil {
		return err
	}
	if err := requireAbsent(destinationDirectory); err != nil {
		return err
	}
	if err := packager.requireCleanRepository(ctx, repositoryRoot); err != nil {
		return err
	}

	checkout, cleanup, err := packager.cloneRepository(ctx, repositoryRoot)
	if err != nil {
		return err
	}
	defer cleanup()

	result, err := packager.Runner.Run(ctx, command.Request{
		Name:      packager.goReleaserBinary(),
		Arguments: append(arguments, "--config", goReleaserConfig),
		Directory: checkout,
		Stdout:    packager.Progress,
		Stderr:    packager.Progress,
	})
	if err != nil {
		return processFailure("build CLI distribution", result, err)
	}
	return publishDirectory(filepath.Join(checkout, "dist"), destinationDirectory)
}

func (packager Packager) requireCleanRepository(ctx context.Context, repositoryRoot string) error {
	result, err := packager.Runner.Run(ctx, command.Request{Name: "git", Arguments: []string{"status", "--porcelain"}, Directory: repositoryRoot})
	if err != nil {
		return processFailure("inspect repository status", result, err)
	}
	if strings.TrimSpace(result.Stdout) != "" {
		return fmt.Errorf("repository contains uncommitted changes; commit or remove them before packaging the CLI")
	}
	return nil
}

func (packager Packager) validatedDestination(destination string) (string, string, error) {
	if packager.Runner == nil {
		return "", "", fmt.Errorf("process runner is not configured")
	}
	if !filepath.IsAbs(destination) {
		return "", "", fmt.Errorf("CLI output path must be absolute")
	}
	repositoryRoot, err := filepath.Abs(packager.RepositoryRoot)
	if err != nil {
		return "", "", fmt.Errorf("resolve repository root: %w", err)
	}
	repositoryRoot, err = filepath.EvalSymlinks(repositoryRoot)
	if err != nil {
		return "", "", fmt.Errorf("resolve repository root: %w", err)
	}
	destinationPath, err := resolvePotentialPath(destination)
	if err != nil {
		return "", "", fmt.Errorf("resolve CLI output path: %w", err)
	}
	if pathWithin(repositoryRoot, destinationPath) {
		return "", "", fmt.Errorf("CLI output path must be outside repository %q", repositoryRoot)
	}
	return repositoryRoot, destinationPath, nil
}

func (packager Packager) cloneRepository(ctx context.Context, repositoryRoot string) (string, func(), error) {
	temporaryRoot, err := os.MkdirTemp("", "rm-relay-distribution-")
	if err != nil {
		return "", func() {}, fmt.Errorf("create temporary distribution root: %w", err)
	}
	cleanup := func() { _ = os.RemoveAll(temporaryRoot) }
	checkout := filepath.Join(temporaryRoot, "repository")
	result, err := packager.Runner.Run(ctx, command.Request{
		Name:      "git",
		Arguments: []string{"clone", "--quiet", "--no-hardlinks", repositoryRoot, checkout},
	})
	if err != nil {
		cleanup()
		return "", func() {}, processFailure("clone committed repository", result, err)
	}
	return checkout, cleanup, nil
}

func (packager Packager) goReleaserBinary() string {
	if packager.GoReleaser != "" {
		return packager.GoReleaser
	}
	return "goreleaser"
}

func publishDirectory(sourceDirectory, destinationDirectory string) error {
	parentDirectory := filepath.Dir(destinationDirectory)
	if err := os.MkdirAll(parentDirectory, 0o755); err != nil {
		return fmt.Errorf("create distribution output parent: %w", err)
	}
	temporaryDirectory, err := os.MkdirTemp(parentDirectory, "."+filepath.Base(destinationDirectory)+".partial-")
	if err != nil {
		return fmt.Errorf("create temporary distribution output: %w", err)
	}
	defer os.RemoveAll(temporaryDirectory)
	if err := copyDirectoryContents(sourceDirectory, temporaryDirectory); err != nil {
		return fmt.Errorf("copy GoReleaser output: %w", err)
	}
	if err := os.Rename(temporaryDirectory, destinationDirectory); err != nil {
		return fmt.Errorf("publish CLI output: %w", err)
	}
	return nil
}

func copyDirectoryContents(sourceRoot, destinationRoot string) error {
	return filepath.WalkDir(sourceRoot, func(sourcePath string, entry fs.DirEntry, walkError error) error {
		if walkError != nil {
			return walkError
		}
		relativePath, err := filepath.Rel(sourceRoot, sourcePath)
		if err != nil {
			return err
		}
		if relativePath == "." {
			return nil
		}
		destinationPath := filepath.Join(destinationRoot, relativePath)
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return os.Mkdir(destinationPath, info.Mode().Perm())
		}
		if entry.Type()&os.ModeSymlink != 0 || !entry.Type().IsRegular() {
			return fmt.Errorf("unsupported generated file %q", relativePath)
		}
		return copyFile(sourcePath, destinationPath, info.Mode().Perm())
	})
}

func copyFile(sourcePath, destinationPath string, mode fs.FileMode) error {
	source, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer source.Close()
	destination, err := os.OpenFile(destinationPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(destination, source); err != nil {
		destination.Close()
		return err
	}
	return destination.Close()
}

func resolvePotentialPath(path string) (string, error) {
	cleanPath := filepath.Clean(path)
	existingPath := cleanPath
	remaining := make([]string, 0, 4)
	for {
		resolved, err := filepath.EvalSymlinks(existingPath)
		if err == nil {
			for index := len(remaining) - 1; index >= 0; index-- {
				resolved = filepath.Join(resolved, remaining[index])
			}
			return resolved, nil
		}
		if !os.IsNotExist(err) {
			return "", err
		}
		parent := filepath.Dir(existingPath)
		if parent == existingPath {
			return "", err
		}
		remaining = append(remaining, filepath.Base(existingPath))
		existingPath = parent
	}
}

func requireAbsent(path string) error {
	if _, err := os.Lstat(path); err == nil {
		return fmt.Errorf("CLI output path already exists: %s", path)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect CLI output path: %w", err)
	}
	return nil
}

func pathWithin(parent, child string) bool {
	relative, err := filepath.Rel(parent, child)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func parseVersion(output string) (string, error) {
	const prefix = "rm-relay version "
	value := strings.TrimSpace(output)
	if !strings.HasPrefix(value, prefix) {
		return "", fmt.Errorf("host CLI returned invalid version output %q", value)
	}
	version := strings.TrimPrefix(value, prefix)
	if version == "" || version == "development" {
		return "", fmt.Errorf("host CLI returned invalid version %q", version)
	}
	return version, nil
}

func fileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open host CLI: %w", err)
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("hash host CLI: %w", err)
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func processFailure(action string, result command.Result, processError error) error {
	details := strings.TrimSpace(result.Stderr)
	if details == "" {
		return fmt.Errorf("%s: %w", action, processError)
	}
	return fmt.Errorf("%s: %w: %s", action, processError, details)
}
