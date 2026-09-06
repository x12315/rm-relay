package builder

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

const (
	// FileName is the workstation-side Builder catalog managed by RM Relay.
	FileName           = "builders.toml"
	storeSchemaVersion = 1
)

type storeDocument struct {
	SchemaVersion int                         `toml:"schema_version"`
	Builders      map[string]storedDefinition `toml:"builders"`
}

type storedDefinition struct {
	Kind          Kind              `toml:"kind"`
	BuildxBuilder string            `toml:"buildx_builder"`
	Environments  map[string]string `toml:"environments"`
}

// Store persists workstation Builder mappings outside user Projects.
type Store struct{ Directory string }

// Load returns an empty catalog when the config has not yet been created.
func (store Store) Load() ([]Definition, error) {
	path := filepath.Join(store.Directory, FileName)
	if info, err := os.Lstat(path); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("Builder catalog must not be a symlink")
	}
	var document storeDocument
	metadata, err := toml.DecodeFile(path, &document)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("decode Builder catalog: %w", err)
	}
	if undecoded := metadata.Undecoded(); len(undecoded) > 0 {
		return nil, fmt.Errorf("Builder catalog contains unknown key %q", undecoded[0].String())
	}
	if document.SchemaVersion != storeSchemaVersion {
		return nil, fmt.Errorf("unsupported Builder catalog schema_version %d", document.SchemaVersion)
	}
	definitions := make([]Definition, 0, len(document.Builders))
	for id, stored := range document.Builders {
		definition := Definition{ID: id, Kind: stored.Kind, BuildxBuilder: stored.BuildxBuilder, Environments: stored.Environments}
		if err := ValidateDefinition(definition); err != nil {
			return nil, err
		}
		definitions = append(definitions, definition)
	}
	return definitions, nil
}

// Save atomically replaces the complete user-managed Builder catalog.
func (store Store) Save(definitions []Definition) error {
	if store.Directory == "" {
		return fmt.Errorf("Builder config directory is not configured")
	}
	if err := os.MkdirAll(store.Directory, 0o700); err != nil {
		return fmt.Errorf("create Builder config directory: %w", err)
	}
	document := storeDocument{SchemaVersion: storeSchemaVersion, Builders: make(map[string]storedDefinition, len(definitions))}
	for _, definition := range definitions {
		if err := ValidateDefinition(definition); err != nil {
			return err
		}
		if _, exists := document.Builders[definition.ID]; exists {
			return fmt.Errorf("multiple builders use ID %q", definition.ID)
		}
		document.Builders[definition.ID] = storedDefinition{Kind: definition.Kind, BuildxBuilder: definition.BuildxBuilder, Environments: definition.Environments}
	}
	temporary, err := os.CreateTemp(store.Directory, ".builders-*.toml")
	if err != nil {
		return fmt.Errorf("create temporary Builder catalog: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return err
	}
	if err := toml.NewEncoder(temporary).Encode(document); err != nil {
		temporary.Close()
		return fmt.Errorf("encode Builder catalog: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return fmt.Errorf("sync Builder catalog: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close Builder catalog: %w", err)
	}
	if err := os.Rename(temporaryPath, filepath.Join(store.Directory, FileName)); err != nil {
		return fmt.Errorf("replace Builder catalog: %w", err)
	}
	return nil
}
