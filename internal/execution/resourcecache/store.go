// Package resourcecache materializes module-owned embedded files for tools that require paths.
package resourcecache

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Store owns an expendable, content-addressed cache below Root.
type Store struct {
	Root string
}

// Materialize writes content atomically and returns a stable path derived from its SHA-256.
func (store Store) Materialize(namespace, name string, content []byte) (string, error) {
	if store.Root == "" {
		return "", fmt.Errorf("resource cache root must not be empty")
	}
	if !safeLogicalPath(namespace) {
		return "", fmt.Errorf("resource namespace %q is not a safe relative path", namespace)
	}
	if !safeFileName(name) {
		return "", fmt.Errorf("resource name %q is not a safe file name", name)
	}
	hash := sha256.Sum256(content)
	directory := filepath.Join(store.Root, filepath.FromSlash(namespace), hex.EncodeToString(hash[:]))
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return "", fmt.Errorf("create resource cache directory: %w", err)
	}
	destination := filepath.Join(directory, name)
	if existing, err := os.ReadFile(destination); err == nil && bytes.Equal(existing, content) {
		return destination, nil
	}

	temporary, err := os.CreateTemp(directory, ".rm-relay-resource-*")
	if err != nil {
		return "", fmt.Errorf("create temporary resource: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o644); err != nil {
		temporary.Close()
		return "", fmt.Errorf("set resource permissions: %w", err)
	}
	if _, err := temporary.Write(content); err != nil {
		temporary.Close()
		return "", fmt.Errorf("write resource: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return "", fmt.Errorf("sync resource: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return "", fmt.Errorf("close resource: %w", err)
	}
	if err := os.Rename(temporaryPath, destination); err != nil {
		if existing, readErr := os.ReadFile(destination); readErr == nil && bytes.Equal(existing, content) {
			return destination, nil
		}
		return "", fmt.Errorf("publish resource: %w", err)
	}
	return destination, nil
}

func safeLogicalPath(value string) bool {
	return value != "" && filepath.IsLocal(filepath.FromSlash(value)) && filepath.Clean(filepath.FromSlash(value)) != "." && !strings.Contains(value, `\`)
}

func safeFileName(value string) bool {
	return value != "" && filepath.Base(value) == value && value != "." && value != ".." && !strings.ContainsAny(value, `/\`)
}
