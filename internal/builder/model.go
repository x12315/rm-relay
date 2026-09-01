// Package builder owns workstation-side build execution resources.
package builder

import (
	"fmt"
	"sort"

	"github.com/x12315/rm-relay/internal/environment"
)

// Kind identifies the execution mechanism behind a logical Builder.
type Kind string

const (
	// KindLocalBuildKit builds through an RM Relay-owned Buildx docker-container resource.
	KindLocalBuildKit Kind = "local-buildkit"
	// KindRemoteBuildKit builds through a named Buildx remote builder.
	KindRemoteBuildKit Kind = "remote-buildkit"
	// LocalID is the built-in logical Builder available without user configuration.
	LocalID = "local"
	// LocalBuildxBuilder is the only workstation Buildx resource owned by RM Relay.
	LocalBuildxBuilder = "rm-relay-local"
	// LocalBuildKitImage pins the BuildKit daemon used by the local Builder.
	LocalBuildKitImage = "moby/buildkit:v0.32.2@sha256:28a898719c18a33f4e8000685287fa36fd0dd9560c6440227d3a732d79bb41d8"
)

// Definition is the resolved workstation resource used for one build.
type Definition struct {
	ID            string
	Kind          Kind
	BuildxBuilder string
	Environments  map[string]string
}

// EnvironmentReference resolves the immutable image selected for a Profile environment.
func (definition Definition) EnvironmentReference(environmentID string) (string, error) {
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
	indexed := map[string]Definition{LocalID: canonicalLocalDefinition()}
	localConfigured := false
	for _, definition := range definitions {
		if err := ValidateDefinition(definition); err != nil {
			return Catalog{}, err
		}
		if definition.ID == LocalID {
			if localConfigured {
				return Catalog{}, fmt.Errorf("multiple builders use ID %q", definition.ID)
			}
			indexed[LocalID] = definition
			localConfigured = true
			continue
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
	if !IsIdentifier(definition.ID) {
		return fmt.Errorf("builder ID %q is invalid", definition.ID)
	}
	if definition.ID == LocalID {
		if definition.Kind != KindLocalBuildKit || definition.BuildxBuilder != LocalBuildxBuilder {
			return fmt.Errorf("local Builder must use kind %q and Buildx resource %q", KindLocalBuildKit, LocalBuildxBuilder)
		}
	} else if definition.Kind != KindRemoteBuildKit {
		return fmt.Errorf("builder %q has unsupported kind %q", definition.ID, definition.Kind)
	}
	if !IsIdentifier(definition.BuildxBuilder) {
		return fmt.Errorf("builder %q has invalid Buildx builder name %q", definition.ID, definition.BuildxBuilder)
	}
	for environmentID, reference := range definition.Environments {
		if !IsIdentifier(environmentID) {
			return fmt.Errorf("builder %q has invalid environment ID %q", definition.ID, environmentID)
		}
		if _, err := environment.ParseDigestReference(reference); err != nil {
			return fmt.Errorf("builder %q environment %q must use an OCI digest reference", definition.ID, environmentID)
		}
	}
	return nil
}

func canonicalLocalDefinition() Definition {
	return Definition{ID: LocalID, Kind: KindLocalBuildKit, BuildxBuilder: LocalBuildxBuilder, Environments: map[string]string{}}
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
