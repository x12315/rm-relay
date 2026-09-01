package builder

import (
	"strings"
	"testing"
)

func TestCatalogAlwaysProvidesLocalBuilder(t *testing.T) {
	catalog, err := NewCatalog()
	if err != nil {
		t.Fatal(err)
	}
	definition, err := catalog.Resolve(LocalID)
	if err != nil {
		t.Fatal(err)
	}
	if definition.Kind != KindLocalBuildKit {
		t.Fatalf("local kind = %q", definition.Kind)
	}
	if definition.BuildxBuilder != LocalBuildxBuilder {
		t.Fatalf("local Buildx builder = %q", definition.BuildxBuilder)
	}
}

func TestCatalogMergesPersistedLocalEnvironmentMappings(t *testing.T) {
	reference := "registry.example/rm-relay/embedded@sha256:" + strings.Repeat("a", 64)
	catalog, err := NewCatalog(Definition{ID: LocalID, Kind: KindLocalBuildKit, BuildxBuilder: LocalBuildxBuilder, Environments: map[string]string{"embedded-development": reference}})
	if err != nil {
		t.Fatal(err)
	}
	definition, err := catalog.Resolve(LocalID)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := definition.EnvironmentReference("embedded-development"); err != nil || got != reference {
		t.Fatalf("EnvironmentReference() = %q, %v", got, err)
	}
}

func TestCatalogRejectsDuplicateLocalDefinitions(t *testing.T) {
	definition := Definition{ID: LocalID, Kind: KindLocalBuildKit, BuildxBuilder: LocalBuildxBuilder}
	if _, err := NewCatalog(definition, definition); err == nil {
		t.Fatal("duplicate local definitions accepted")
	}
}
