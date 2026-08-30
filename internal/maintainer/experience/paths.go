// Package experience owns the disposable local environment used to review one RM Relay candidate.
package experience

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// Layout identifies every path owned by one repository's candidate experience environment.
type Layout struct {
	RepositoryRoot  string
	RepositoryKey   string
	Root            string
	StatePath       string
	BinaryDirectory string
	BinaryPath      string
	TemplateOrigin  string
	Workspace       string
	Logs            string
}

// ResolveLayout maps one canonical repository to its platform user-cache location.
func ResolveLayout(repositoryRoot, userCacheRoot string) (Layout, error) {
	canonicalRepository, err := canonicalDirectory(repositoryRoot)
	if err != nil {
		return Layout{}, fmt.Errorf("resolve repository root: %w", err)
	}
	cacheRoot, err := filepath.Abs(userCacheRoot)
	if err != nil {
		return Layout{}, fmt.Errorf("resolve user cache root: %w", err)
	}
	if cacheRoot == canonicalRepository {
		return Layout{}, fmt.Errorf("user cache root must not equal repository root")
	}
	digest := sha256.Sum256([]byte(canonicalRepository))
	repositoryKey := hex.EncodeToString(digest[:8])
	root := filepath.Join(filepath.Clean(cacheRoot), "rm-relay", "experience", repositoryKey)
	binaryDirectory := filepath.Join(root, "bin")
	binaryName := "rm-relay"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	return Layout{
		RepositoryRoot:  canonicalRepository,
		RepositoryKey:   repositoryKey,
		Root:            root,
		StatePath:       filepath.Join(root, "state.json"),
		BinaryDirectory: binaryDirectory,
		BinaryPath:      filepath.Join(binaryDirectory, binaryName),
		TemplateOrigin:  filepath.Join(root, "template.git"),
		Workspace:       filepath.Join(root, "workspace"),
		Logs:            filepath.Join(root, "logs"),
	}, nil
}

func canonicalDirectory(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path must not be empty")
	}
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	resolvedPath, err := filepath.EvalSymlinks(absolutePath)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(resolvedPath)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%q is not a directory", resolvedPath)
	}
	return filepath.Clean(resolvedPath), nil
}
