// Package project owns the versioned declaration stored in a user project.
package project

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"
)

const (
	// FileName is the project declaration discovered at a project root.
	FileName = "rm-relay.toml"

	currentSchemaVersion = 1
)

var blankProjectIDAssignment = regexp.MustCompile(`(?m)^([ \t]*project_id[ \t]*=[ \t]*)""([ \t]*(?:#.*)?)(\r?)$`)

// Config is the versioned declaration that binds a project to supported builds.
type Config struct {
	SchemaVersion  int     `toml:"schema_version"`
	ProjectID      string  `toml:"project_id"`
	DefaultProfile string  `toml:"default_profile"`
	Builds         []Build `toml:"builds"`
}

// Build declares how one RM Relay Profile maps to a project build-system entry.
type Build struct {
	Profile string   `toml:"profile"`
	System  string   `toml:"system"`
	Preset  string   `toml:"preset"`
	Outputs []Output `toml:"outputs"`
}

// Output maps a semantic artifact role to a path below the profile install directory.
type Output struct {
	Role string `toml:"role"`
	Path string `toml:"path"`
}

// Load parses and validates the project declaration at root.
// An empty ProjectID is valid only so that Initialize can assign it.
func Load(root string) (Config, error) {
	var config Config
	configPath := filepath.Join(root, FileName)
	metadata, err := toml.DecodeFile(configPath, &config)
	if err != nil {
		return Config{}, fmt.Errorf("decode %s: %w", configPath, err)
	}
	if undecoded := metadata.Undecoded(); len(undecoded) > 0 {
		return Config{}, fmt.Errorf("%s contains unknown key %q", FileName, undecoded[0].String())
	}
	if err := config.validate(); err != nil {
		return Config{}, fmt.Errorf("validate %s: %w", FileName, err)
	}
	return config, nil
}

// BuildForProfile returns the single build declaration bound to profileID.
func (config Config) BuildForProfile(profileID string) (Build, error) {
	var matches []Build
	for _, build := range config.Builds {
		if build.Profile == profileID {
			matches = append(matches, build)
		}
	}
	switch len(matches) {
	case 0:
		return Build{}, fmt.Errorf("project does not declare profile %q", profileID)
	case 1:
		return matches[0], nil
	default:
		return Build{}, fmt.Errorf("project contains multiple build declarations for profile %q", profileID)
	}
}

// Initialize assigns a UUID v4 to a declaration whose ProjectID is blank.
// Existing project identities are returned without rewriting the file.
func Initialize(root string) (string, error) {
	config, err := Load(root)
	if err != nil {
		return "", err
	}
	if config.ProjectID != "" {
		return config.ProjectID, nil
	}

	configPath := filepath.Join(root, FileName)
	original, err := os.ReadFile(configPath)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", configPath, err)
	}
	matches := blankProjectIDAssignment.FindAllSubmatchIndex(original, -1)
	if len(matches) != 1 {
		return "", fmt.Errorf("%s must contain exactly one blank project_id assignment", FileName)
	}
	projectID, err := newUUIDv4()
	if err != nil {
		return "", err
	}
	match := matches[0]
	updated := make([]byte, 0, len(original)+len(projectID))
	updated = append(updated, original[:match[2]]...)
	updated = append(updated, original[match[2]:match[3]]...)
	updated = append(updated, '"')
	updated = append(updated, projectID...)
	updated = append(updated, '"')
	updated = append(updated, original[match[4]:match[5]]...)
	updated = append(updated, original[match[1]:]...)

	fileInfo, err := os.Stat(configPath)
	if err != nil {
		return "", fmt.Errorf("stat %s: %w", configPath, err)
	}
	temporaryFile, err := os.CreateTemp(root, ".rm-relay.toml-*")
	if err != nil {
		return "", fmt.Errorf("create temporary project config: %w", err)
	}
	temporaryPath := temporaryFile.Name()
	defer os.Remove(temporaryPath)
	if err := temporaryFile.Chmod(fileInfo.Mode().Perm()); err != nil {
		temporaryFile.Close()
		return "", fmt.Errorf("preserve project config permissions: %w", err)
	}
	if _, err := temporaryFile.Write(updated); err != nil {
		temporaryFile.Close()
		return "", fmt.Errorf("write initialized project config: %w", err)
	}
	if err := temporaryFile.Sync(); err != nil {
		temporaryFile.Close()
		return "", fmt.Errorf("sync initialized project config: %w", err)
	}
	if err := temporaryFile.Close(); err != nil {
		return "", fmt.Errorf("close initialized project config: %w", err)
	}
	if err := os.Rename(temporaryPath, configPath); err != nil {
		return "", fmt.Errorf("replace project config: %w", err)
	}
	return projectID, nil
}

func (config Config) validate() error {
	if config.SchemaVersion != currentSchemaVersion {
		return fmt.Errorf("unsupported schema_version %d", config.SchemaVersion)
	}
	if config.ProjectID != "" && !isUUIDv4(config.ProjectID) {
		return fmt.Errorf("project_id %q is not a UUID v4", config.ProjectID)
	}
	if config.DefaultProfile == "" {
		return fmt.Errorf("default_profile must not be empty")
	}
	if len(config.Builds) == 0 {
		return fmt.Errorf("at least one build declaration is required")
	}
	profiles := make(map[string]struct{}, len(config.Builds))
	for buildIndex, build := range config.Builds {
		if build.Profile == "" {
			return fmt.Errorf("build %d profile must not be empty", buildIndex)
		}
		if _, exists := profiles[build.Profile]; exists {
			return fmt.Errorf("multiple build declarations use profile %q", build.Profile)
		}
		profiles[build.Profile] = struct{}{}
		if !isIdentifier(build.System) {
			return fmt.Errorf("build %q system %q is not a valid identifier", build.Profile, build.System)
		}
		if !isIdentifier(build.Preset) {
			return fmt.Errorf("build %q preset %q is not a valid identifier", build.Profile, build.Preset)
		}
		if len(build.Outputs) == 0 {
			return fmt.Errorf("build %q must declare outputs", build.Profile)
		}
		roles := make(map[string]struct{}, len(build.Outputs))
		for _, output := range build.Outputs {
			if strings.TrimSpace(output.Role) == "" {
				return fmt.Errorf("build %q contains an empty output role", build.Profile)
			}
			if _, exists := roles[output.Role]; exists {
				return fmt.Errorf("build %q contains duplicate output role %q", build.Profile, output.Role)
			}
			roles[output.Role] = struct{}{}
			if !isSafeRelativeFile(output.Path) {
				return fmt.Errorf("build %q output path %q is not a safe relative file", build.Profile, output.Path)
			}
		}
	}
	if _, exists := profiles[config.DefaultProfile]; !exists {
		return fmt.Errorf("default_profile %q has no build declaration", config.DefaultProfile)
	}
	return nil
}

func isIdentifier(value string) bool {
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

func newUUIDv4() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", fmt.Errorf("generate project identity: %w", err)
	}
	value[6] = value[6]&0x0f | 0x40
	value[8] = value[8]&0x3f | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		value[0:4], value[4:6], value[6:8], value[8:10], value[10:16]), nil
}

func isUUIDv4(value string) bool {
	if len(value) != 36 || value[8] != '-' || value[13] != '-' || value[18] != '-' || value[23] != '-' || value[14] != '4' {
		return false
	}
	if !strings.ContainsRune("89abAB", rune(value[19])) {
		return false
	}
	for index, character := range value {
		if index == 8 || index == 13 || index == 18 || index == 23 {
			continue
		}
		if !strings.ContainsRune("0123456789abcdefABCDEF", character) {
			return false
		}
	}
	return true
}
