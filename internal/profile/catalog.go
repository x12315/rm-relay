// Package profile owns RM Relay's versioned catalog of supported development combinations.
package profile

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/BurntSushi/toml"
)

const (
	// FileName is the manifest stored in each Profile catalog directory.
	FileName = "profile.toml"

	currentSchemaVersion = 1
)

//go:embed builtin/*/profile.toml
var builtinFiles embed.FS

// Catalog reads Profile definitions from one logical filesystem.
// Its fs.FS boundary permits built-in, test and future external catalogs to share validation.
type Catalog struct {
	Files fs.FS
	Root  string
}

// Config is the versioned Profile manifest.
type Config struct {
	SchemaVersion       int               `toml:"schema_version"`
	ID                  string            `toml:"id"`
	DevelopmentImage    string            `toml:"development_image"`
	RequiredOutputRoles []string          `toml:"required_output_roles"`
	Targets             map[string]Target `toml:"targets"`
}

// Target binds a semantic target capability to an adapter-owned board definition.
type Target struct {
	Adapter      string `toml:"adapter"`
	Board        string `toml:"board"`
	ArtifactRole string `toml:"artifact_role"`
}

// Loaded combines a validated Profile with its content identity.
type Loaded struct {
	Config Config
	Digest string
}

// BuiltinCatalog returns the Profile catalog shipped in the rm-relay executable.
func BuiltinCatalog() Catalog {
	return Catalog{Files: builtinFiles, Root: "builtin"}
}

// Load validates and identifies one Profile from the catalog.
func (catalog Catalog) Load(id string) (Loaded, error) {
	if !isCatalogKey(id) {
		return Loaded{}, fmt.Errorf("profile ID %q is not a safe catalog key", id)
	}
	if catalog.Files == nil {
		return Loaded{}, fmt.Errorf("profile catalog filesystem is not configured")
	}
	manifestPath := path.Join(catalog.Root, id, FileName)
	content, err := fs.ReadFile(catalog.Files, manifestPath)
	if err != nil {
		return Loaded{}, fmt.Errorf("read profile %q: %w", id, err)
	}

	var config Config
	metadata, err := toml.Decode(string(content), &config)
	if err != nil {
		return Loaded{}, fmt.Errorf("decode profile %q: %w", id, err)
	}
	if undecoded := metadata.Undecoded(); len(undecoded) > 0 {
		return Loaded{}, fmt.Errorf("profile %q contains unknown key %q", id, undecoded[0].String())
	}
	if err := validateConfig(id, config); err != nil {
		return Loaded{}, err
	}

	hash := sha256.New()
	hash.Write([]byte(FileName))
	hash.Write([]byte{0})
	hash.Write(content)
	return Loaded{Config: config, Digest: hex.EncodeToString(hash.Sum(nil))}, nil
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
		if !isCatalogKey(target.Adapter) {
			return fmt.Errorf("profile %q target %q adapter %q is invalid", requestedID, targetName, target.Adapter)
		}
		if !isCatalogKey(target.Board) {
			return fmt.Errorf("profile %q target %q board %q is invalid", requestedID, targetName, target.Board)
		}
		if _, exists := requiredRoles[target.ArtifactRole]; !exists {
			return fmt.Errorf("profile %q target %q consumes undeclared role %q", requestedID, targetName, target.ArtifactRole)
		}
	}
	return nil
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
