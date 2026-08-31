// Package builder owns workstation-side build execution resources.
package builder

import (
	"fmt"
	"sort"
)

// Kind identifies the execution mechanism behind a logical Builder.
type Kind string

const (
	// KindLocalContainer builds through the workstation Docker Engine.
	KindLocalContainer Kind = "local-container"
	// KindRemoteBuildKit builds through a named Buildx remote builder.
	KindRemoteBuildKit Kind = "remote-buildkit"
	// LocalID is the built-in logical Builder available without user configuration.
	LocalID = "local"
)

// Definition is the resolved workstation resource used for one build.
type Definition struct {
	ID            string
	Kind          Kind
	BuildxBuilder string
	Environments  map[string]string
}

// EnvironmentReference resolves the immutable or local image reference for a Profile environment.
func (definition Definition) EnvironmentReference(environmentID, localReference string) (string, error) {
	if definition.Kind == KindLocalContainer {
		if localReference == "" {
			return "", fmt.Errorf("environment %q has no local reference", environmentID)
		}
		return localReference, nil
	}
	reference := definition.Environments[environmentID]
	if reference == "" {
		return "", fmt.Errorf("builder %q has no mapping for environment %q", definition.ID, environmentID)
	}
	return reference, nil
}

// Catalog combines the built-in local Builder with user-configured definitions.
type Catalog struct {
	definitions map[string]Definition
}

// NewCatalog validates and indexes definitions. The built-in local Builder is always present.
func NewCatalog(definitions ...Definition) (Catalog, error) {
	indexed := map[string]Definition{LocalID: {ID: LocalID, Kind: KindLocalContainer}}
	for _, definition := range definitions {
		if err := ValidateDefinition(definition); err != nil {
			return Catalog{}, err
		}
		if _, exists := indexed[definition.ID]; exists {
			return Catalog{}, fmt.Errorf("multiple builders use ID %q", definition.ID)
		}
		indexed[definition.ID] = definition
	}
	return Catalog{definitions: indexed}, nil
}

// Resolve returns one logical Builder by ID.
func (catalog Catalog) Resolve(id string) (Definition, error) {
	definition, exists := catalog.definitions[id]
	if exists {
		return definition, nil
	}
	return Definition{}, fmt.Errorf("unknown builder %q; available builders: %v", id, catalog.IDs())
}

// IDs returns stable sorted logical Builder IDs.
func (catalog Catalog) IDs() []string {
	ids := make([]string, 0, len(catalog.definitions))
	for id := range catalog.definitions {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// ValidateDefinition checks persistent Builder invariants.
func ValidateDefinition(definition Definition) error {
	if !IsIdentifier(definition.ID) || definition.ID == LocalID {
		return fmt.Errorf("builder ID %q is invalid or reserved", definition.ID)
	}
	if definition.Kind != KindRemoteBuildKit {
		return fmt.Errorf("builder %q has unsupported kind %q", definition.ID, definition.Kind)
	}
	if !IsIdentifier(definition.BuildxBuilder) {
		return fmt.Errorf("builder %q has invalid Buildx builder name %q", definition.ID, definition.BuildxBuilder)
	}
	for environmentID, reference := range definition.Environments {
		if !IsIdentifier(environmentID) {
			return fmt.Errorf("builder %q has invalid environment ID %q", definition.ID, environmentID)
		}
		if !IsDigestReference(reference) {
			return fmt.Errorf("builder %q environment %q must use an OCI digest reference", definition.ID, environmentID)
		}
	}
	return nil
}

// IsIdentifier reports whether value is a stable RM Relay identifier.
func IsIdentifier(value string) bool {
	if value == "" {
		return false
	}
	for index, character := range value {
		if (character >= 'a' && character <= 'z') || (character >= 'A' && character <= 'Z') ||
			(character >= '0' && character <= '9') || (index > 0 && (character == '-' || character == '_' || character == '.')) {
			continue
		}
		return false
	}
	return true
}

// IsDigestReference requires an OCI reference pinned to a SHA-256 digest.
func IsDigestReference(reference string) bool {
	const marker = "@sha256:"
	for index := 0; index+len(marker) <= len(reference); index++ {
		if reference[index:index+len(marker)] != marker {
			continue
		}
		digest := reference[index+len(marker):]
		if index == 0 || len(digest) != 64 {
			return false
		}
		for _, character := range digest {
			if (character < '0' || character > '9') && (character < 'a' || character > 'f') {
				return false
			}
		}
		return true
	}
	return false
}
