package main

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/x12315/rm-relay/internal/builder"
)

func TestResolveCandidateBuilderUsesBuiltInLocalDefinition(t *testing.T) {
	definition, err := resolveCandidateBuilder(t.TempDir(), builder.LocalID)
	if err != nil {
		t.Fatal(err)
	}
	if definition.ID != builder.LocalID || definition.Kind != builder.KindLocalBuildKit || definition.BuildxBuilder != builder.LocalBuildxBuilder {
		t.Fatalf("local Builder = %#v", definition)
	}
}

func TestResolveCandidateBuilderReadsConfiguredRemoteDefinition(t *testing.T) {
	configRoot := t.TempDir()
	want := builder.Definition{
		ID:            "team",
		Kind:          builder.KindRemoteBuildKit,
		BuildxBuilder: "rm-relay-team",
		Environments: map[string]string{
			"embedded-development": "registry.example/environment@sha256:" + strings.Repeat("a", 64),
		},
	}
	store := builder.Store{Directory: filepath.Join(configRoot, "rm-relay")}
	if err := store.Save([]builder.Definition{want}); err != nil {
		t.Fatal(err)
	}

	got, err := resolveCandidateBuilder(configRoot, want.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != want.ID || got.Kind != want.Kind || got.BuildxBuilder != want.BuildxBuilder {
		t.Fatalf("remote Builder = %#v, want %#v", got, want)
	}
}
