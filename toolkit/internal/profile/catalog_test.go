package profile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCatalogRejectsRequestedIDDifferentFromManifest(t *testing.T) {
	assetsRoot := writeProfileAssets(t, "requested-profile", "different-profile", "openocd/board.cfg")

	_, err := testCatalog(assetsRoot).Load("requested-profile")
	if err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("Load() error = %v, want profile ID mismatch", err)
	}
}

func TestCatalogRejectsTargetConfigOutsideAssetRoot(t *testing.T) {
	assetsRoot := writeProfileAssets(t, "embedded-test", "embedded-test", "../outside.cfg")
	if err := os.WriteFile(filepath.Join(assetsRoot, "..", "outside.cfg"), []byte("adapter speed 1800\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := testCatalog(assetsRoot).Load("embedded-test")
	if err == nil || !strings.Contains(err.Error(), "target config") {
		t.Fatalf("Load() error = %v, want target config boundary error", err)
	}
}

func TestCatalogDigestChangesWhenExecutionAssetChanges(t *testing.T) {
	assetsRoot := writeProfileAssets(t, "embedded-test", "embedded-test", "openocd/board.cfg")
	catalog := testCatalog(assetsRoot)

	before, err := catalog.Load("embedded-test")
	if err != nil {
		t.Fatalf("first Load() error = %v", err)
	}
	configPath := filepath.Join(assetsRoot, "openocd", "board.cfg")
	if err := os.WriteFile(configPath, []byte("adapter speed 2400\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	after, err := catalog.Load("embedded-test")
	if err != nil {
		t.Fatalf("second Load() error = %v", err)
	}
	if before.Digest == after.Digest {
		t.Fatalf("profile digest did not change after %s changed", configPath)
	}
}

func TestCatalogLoadsRequiredOutputRolesAndOpenOCDTarget(t *testing.T) {
	assetsRoot := writeProfileAssets(t, "embedded-test", "embedded-test", "openocd/board.cfg")

	loaded, err := testCatalog(assetsRoot).Load("embedded-test")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.Config.DevelopmentImage != "mcu-dev/toolchain:test" {
		t.Fatalf("DevelopmentImage = %q", loaded.Config.DevelopmentImage)
	}
	if len(loaded.Config.RequiredOutputRoles) != 3 {
		t.Fatalf("RequiredOutputRoles = %v", loaded.Config.RequiredOutputRoles)
	}
	target, found := loaded.Config.Targets["openocd-stlink"]
	if !found || target.Adapter != "openocd" || target.ArtifactRole != "firmware.elf" {
		t.Fatalf("openocd-stlink target = %+v, found = %v", target, found)
	}
	if len(loaded.Digest) != 64 {
		t.Fatalf("Digest = %q, want SHA-256 hex", loaded.Digest)
	}
}

func testCatalog(assetsRoot string) Catalog {
	return Catalog{
		ProfilesRoot: filepath.Join(assetsRoot, "profiles"),
		AssetsRoot:   assetsRoot,
	}
}

func writeProfileAssets(t *testing.T, directoryID, manifestID, targetConfig string) string {
	t.Helper()
	assetsRoot := t.TempDir()
	profileDirectory := filepath.Join(assetsRoot, "profiles", directoryID)
	if err := os.MkdirAll(profileDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(assetsRoot, "openocd"), 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `schema_version = 1
id = "` + manifestID + `"
development_image = "mcu-dev/toolchain:test"
mise_config = "mise.toml"
required_output_roles = ["firmware.elf", "firmware.bin", "linker.map"]

[targets.openocd-stlink]
adapter = "openocd"
config = "` + targetConfig + `"
artifact_role = "firmware.elf"
`
	if err := os.WriteFile(filepath.Join(profileDirectory, FileName), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(profileDirectory, "mise.toml"), []byte("min_version = \"2026.8.14\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if targetConfig == "openocd/board.cfg" {
		if err := os.WriteFile(filepath.Join(assetsRoot, targetConfig), []byte("adapter speed 1800\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return assetsRoot
}
