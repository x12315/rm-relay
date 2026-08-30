package profile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func TestCatalogRejectsRequestedIDDifferentFromManifest(t *testing.T) {
	assetsRoot := writeProfileAssets(t, "requested-profile", "different-profile", "openocd/board.cfg")

	_, err := testCatalog(assetsRoot).Load("requested-profile")
	if err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("Load() error = %v, want profile ID mismatch", err)
	}
}

func TestCatalogDigestChangesWhenProfileManifestChanges(t *testing.T) {
	assetsRoot := writeProfileAssets(t, "embedded-test", "embedded-test", "robomaster-c")
	catalog := testCatalog(assetsRoot)

	before, err := catalog.Load("embedded-test")
	if err != nil {
		t.Fatalf("first Load() error = %v", err)
	}
	manifestPath := filepath.Join(assetsRoot, "profiles", "embedded-test", FileName)
	content, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, []byte(strings.Replace(string(content), "mcu-dev/toolchain:test", "mcu-dev/toolchain:changed", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	after, err := catalog.Load("embedded-test")
	if err != nil {
		t.Fatalf("second Load() error = %v", err)
	}
	if before.Digest == after.Digest {
		t.Fatalf("profile digest did not change after %s changed", manifestPath)
	}
}

func TestCatalogLoadsRequiredOutputRolesAndOpenOCDTarget(t *testing.T) {
	assetsRoot := writeProfileAssets(t, "embedded-test", "embedded-test", "robomaster-c")

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
	if !found || target.Adapter != "openocd" || target.Board != "robomaster-c" || target.ArtifactRole != "firmware.elf" {
		t.Fatalf("openocd-stlink target = %+v, found = %v", target, found)
	}
	if len(loaded.Digest) != 64 {
		t.Fatalf("Digest = %q, want SHA-256 hex", loaded.Digest)
	}
}

func testCatalog(assetsRoot string) Catalog {
	return Catalog{
		Files: os.DirFS(assetsRoot),
		Root:  "profiles",
	}
}

func TestBuiltinCatalogProvidesSupportedProfile(t *testing.T) {
	loaded, err := BuiltinCatalog().Load("embedded-stm32f407-robomaster-c")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.Config.Targets["openocd-stlink"].Board != "robomaster-c" {
		t.Fatalf("builtin OpenOCD board = %q", loaded.Config.Targets["openocd-stlink"].Board)
	}
}

func TestCatalogCanUseAnExternalFilesystemWithoutChangingConsumers(t *testing.T) {
	catalog := Catalog{Files: fstest.MapFS{
		"profiles/external/profile.toml": {Data: []byte(`schema_version = 1
id = "external"
development_image = "example.invalid/development:1"
required_output_roles = ["application"]
`)},
	}, Root: "profiles"}

	loaded, err := catalog.Load("external")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.Config.ID != "external" {
		t.Fatalf("profile ID = %q", loaded.Config.ID)
	}
}

func TestCatalogDigestDoesNotDependOnCatalogLocation(t *testing.T) {
	manifest := []byte(`schema_version = 1
id = "portable"
development_image = "example.invalid/development:1"
required_output_roles = ["application"]
`)
	files := fstest.MapFS{
		"installed/profiles/portable/profile.toml":   {Data: manifest},
		"checked-out/profiles/portable/profile.toml": {Data: manifest},
	}

	installed, err := (Catalog{Files: files, Root: "installed/profiles"}).Load("portable")
	if err != nil {
		t.Fatalf("load installed catalog: %v", err)
	}
	checkedOut, err := (Catalog{Files: files, Root: "checked-out/profiles"}).Load("portable")
	if err != nil {
		t.Fatalf("load checked-out catalog: %v", err)
	}
	if installed.Digest != checkedOut.Digest {
		t.Fatalf("same profile has location-dependent digests: %q != %q", installed.Digest, checkedOut.Digest)
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
required_output_roles = ["firmware.elf", "firmware.bin", "linker.map"]

[targets.openocd-stlink]
adapter = "openocd"
board = "` + targetConfig + `"
artifact_role = "firmware.elf"
`
	if err := os.WriteFile(filepath.Join(profileDirectory, FileName), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	return assetsRoot
}
