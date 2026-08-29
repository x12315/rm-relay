// Package profile owns RM Relay's versioned catalog of supported development combinations.
package profile

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
)

const (
	// FileName is the manifest stored in each Profile catalog directory.
	FileName = "profile.toml"

	currentSchemaVersion = 1
)

// Catalog locates Profile definitions within an RM Relay asset tree.
type Catalog struct {
	ProfilesRoot string
	AssetsRoot   string
}

// Config is the versioned Profile manifest.
type Config struct {
	SchemaVersion       int               `toml:"schema_version"`
	ID                  string            `toml:"id"`
	DevelopmentImage    string            `toml:"development_image"`
	MiseConfig          string            `toml:"mise_config"`
	RequiredOutputRoles []string          `toml:"required_output_roles"`
	Targets             map[string]Target `toml:"targets"`
}

// Target declares one development-machine adapter exposed by a Profile.
type Target struct {
	Adapter      string `toml:"adapter"`
	Config       string `toml:"config"`
	ArtifactRole string `toml:"artifact_role"`
}

// Loaded combines a validated Profile with its resolved location and content identity.
type Loaded struct {
	Config     Config
	Directory  string
	AssetsRoot string
	Digest     string
}

// Load validates and identifies one Profile from the catalog.
func (catalog Catalog) Load(id string) (Loaded, error) {
	if !isCatalogKey(id) {
		return Loaded{}, fmt.Errorf("profile ID %q is not a safe catalog key", id)
	}
	profilesRoot, err := filepath.Abs(catalog.ProfilesRoot)
	if err != nil {
		return Loaded{}, fmt.Errorf("resolve profiles root: %w", err)
	}
	assetsRoot, err := filepath.Abs(catalog.AssetsRoot)
	if err != nil {
		return Loaded{}, fmt.Errorf("resolve assets root: %w", err)
	}
	profileDirectory := filepath.Join(profilesRoot, id)
	manifestPath := filepath.Join(profileDirectory, FileName)

	var config Config
	metadata, err := toml.DecodeFile(manifestPath, &config)
	if err != nil {
		return Loaded{}, fmt.Errorf("decode profile %q: %w", id, err)
	}
	if undecoded := metadata.Undecoded(); len(undecoded) > 0 {
		return Loaded{}, fmt.Errorf("profile %q contains unknown key %q", id, undecoded[0].String())
	}
	if err := validateConfig(id, config); err != nil {
		return Loaded{}, err
	}

	executionAssets := map[string]string{
		filepath.ToSlash(filepath.Join("profiles", id, FileName)): manifestPath,
	}
	misePath, err := resolveRegularFile(profileDirectory, config.MiseConfig)
	if err != nil {
		return Loaded{}, fmt.Errorf("profile %q mise config: %w", id, err)
	}
	executionAssets[filepath.ToSlash(filepath.Join("profiles", id, config.MiseConfig))] = misePath
	for targetName, target := range config.Targets {
		targetConfigPath, err := resolveRegularFile(assetsRoot, target.Config)
		if err != nil {
			return Loaded{}, fmt.Errorf("profile %q target config %q: %w", id, targetName, err)
		}
		executionAssets[filepath.ToSlash(filepath.Clean(target.Config))] = targetConfigPath
	}
	digest, err := digestFiles(executionAssets)
	if err != nil {
		return Loaded{}, fmt.Errorf("digest profile %q: %w", id, err)
	}
	return Loaded{
		Config:     config,
		Directory:  profileDirectory,
		AssetsRoot: assetsRoot,
		Digest:     digest,
	}, nil
}

func validateConfig(requestedID string, config Config) error {
	if config.SchemaVersion != currentSchemaVersion {
		return fmt.Errorf("profile %q has unsupported schema_version %d", requestedID, config.SchemaVersion)
	}
	if config.ID != requestedID {
		return fmt.Errorf("profile manifest ID %q does not match requested ID %q", config.ID, requestedID)
	}
	if strings.TrimSpace(config.DevelopmentImage) == "" {
		return fmt.Errorf("profile %q development_image must not be empty", requestedID)
	}
	if !isSafeRelativeFile(config.MiseConfig) {
		return fmt.Errorf("profile %q mise_config must be a safe relative file", requestedID)
	}
	if len(config.RequiredOutputRoles) == 0 {
		return fmt.Errorf("profile %q must require at least one output role", requestedID)
	}
	requiredRoles := make(map[string]struct{}, len(config.RequiredOutputRoles))
	for _, role := range config.RequiredOutputRoles {
		if strings.TrimSpace(role) == "" {
			return fmt.Errorf("profile %q contains an empty required output role", requestedID)
		}
		if _, exists := requiredRoles[role]; exists {
			return fmt.Errorf("profile %q contains duplicate required output role %q", requestedID, role)
		}
		requiredRoles[role] = struct{}{}
	}
	for targetName, target := range config.Targets {
		if !isCatalogKey(targetName) {
			return fmt.Errorf("profile %q target name %q is invalid", requestedID, targetName)
		}
		if target.Adapter != "openocd" {
			return fmt.Errorf("profile %q target %q uses unsupported adapter %q", requestedID, targetName, target.Adapter)
		}
		if !isSafeRelativeFile(target.Config) {
			return fmt.Errorf("profile %q target config %q is not a safe relative file", requestedID, target.Config)
		}
		if _, exists := requiredRoles[target.ArtifactRole]; !exists {
			return fmt.Errorf("profile %q target %q consumes undeclared role %q", requestedID, targetName, target.ArtifactRole)
		}
	}
	return nil
}

func digestFiles(files map[string]string) (string, error) {
	logicalPaths := make([]string, 0, len(files))
	for logicalPath := range files {
		logicalPaths = append(logicalPaths, logicalPath)
	}
	sort.Strings(logicalPaths)

	hash := sha256.New()
	for _, logicalPath := range logicalPaths {
		content, err := os.ReadFile(files[logicalPath])
		if err != nil {
			return "", fmt.Errorf("read %s: %w", logicalPath, err)
		}
		hash.Write([]byte(logicalPath))
		hash.Write([]byte{0})
		hash.Write(content)
		hash.Write([]byte{0})
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func resolveRegularFile(root, relativePath string) (string, error) {
	if !isSafeRelativeFile(relativePath) {
		return "", fmt.Errorf("path %q escapes its asset root", relativePath)
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	resolvedPath := filepath.Join(root, filepath.Clean(relativePath))
	relativeToRoot, err := filepath.Rel(root, resolvedPath)
	if err != nil || !filepath.IsLocal(relativeToRoot) {
		return "", fmt.Errorf("path %q escapes its asset root", relativePath)
	}
	fileInfo, err := os.Lstat(resolvedPath)
	if err != nil {
		return "", err
	}
	if fileInfo.Mode()&os.ModeSymlink != 0 || !fileInfo.Mode().IsRegular() {
		return "", fmt.Errorf("path %q must name a regular file", relativePath)
	}
	return resolvedPath, nil
}

func isCatalogKey(value string) bool {
	if value == "" {
		return false
	}
	for index, character := range value {
		if (character >= 'a' && character <= 'z') ||
			(character >= 'A' && character <= 'Z') ||
			(character >= '0' && character <= '9') ||
			(index > 0 && (character == '-' || character == '_' || character == '.')) {
			continue
		}
		return false
	}
	return true
}

func isSafeRelativeFile(path string) bool {
	return filepath.IsLocal(path) && filepath.Clean(path) != "."
}
