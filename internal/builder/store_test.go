package builder

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStoreRoundTripKeepsOnlyLogicalMappings(t *testing.T) {
	store := Store{Directory: filepath.Join(t.TempDir(), "config")}
	definition := Definition{ID: "team", Kind: KindRemoteBuildKit, BuildxBuilder: "rm-relay-team", Environments: map[string]string{"embedded-development": "registry.example/image@sha256:" + strings.Repeat("a", 64)}}
	if err := store.Save([]Definition{definition}); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 1 || loaded[0].BuildxBuilder != definition.BuildxBuilder {
		t.Fatalf("loaded = %#v", loaded)
	}
	info, err := os.Stat(filepath.Join(store.Directory, FileName))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("catalog mode = %o", info.Mode().Perm())
	}
	content, _ := os.ReadFile(filepath.Join(store.Directory, FileName))
	if strings.Contains(string(content), "cert") || strings.Contains(string(content), "key.pem") {
		t.Fatalf("catalog persisted credentials:\n%s", content)
	}
}

func TestStoreAcceptsOnlyCanonicalLocalDefinition(t *testing.T) {
	store := Store{Directory: filepath.Join(t.TempDir(), "config")}
	canonical := Definition{ID: LocalID, Kind: KindLocalBuildKit, BuildxBuilder: LocalBuildxBuilder, Environments: map[string]string{}}
	if err := store.Save([]Definition{canonical}); err != nil {
		t.Fatal(err)
	}
	canonical.BuildxBuilder = "foreign-local"
	if err := store.Save([]Definition{canonical}); err == nil {
		t.Fatal("noncanonical local Builder accepted")
	}
}

func TestStoreRejectsUnknownKeysAndSymlinkCatalog(t *testing.T) {
	root := t.TempDir()
	store := Store{Directory: root}
	if err := os.WriteFile(filepath.Join(root, FileName), []byte("schema_version = 1\nunknown = true\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Load(); err == nil {
		t.Fatal("unknown key accepted")
	}
	target := filepath.Join(root, "target.toml")
	if err := os.WriteFile(target, []byte("schema_version = 1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(root, FileName)); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(root, FileName)); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err := store.Load(); err == nil {
		t.Fatal("symlink catalog accepted")
	}
}
