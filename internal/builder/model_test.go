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
	if definition.Kind != KindLocalContainer {
		t.Fatalf("local kind = %q", definition.Kind)
	}
}

func TestDigestReferenceRejectsMutableTags(t *testing.T) {
	if IsDigestReference("registry.example/image:latest") {
		t.Fatal("mutable tag accepted")
	}
	if !IsDigestReference("registry.example/image@sha256:" + strings.Repeat("a", 64)) {
		t.Fatal("valid digest reference rejected")
	}
}
