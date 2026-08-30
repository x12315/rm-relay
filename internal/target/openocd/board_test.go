package openocd

import (
	"os"
	"strings"
	"testing"

	"github.com/x12315/rm-relay/internal/execution/resourcecache"
)

func TestBuiltinBoardCatalogMaterializesRoboMasterC(t *testing.T) {
	catalog := BuiltinBoardCatalog(resourcecache.Store{Root: t.TempDir()})
	configPath, err := catalog.Resolve("robomaster-c")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "target/stm32f4x.cfg") {
		t.Fatalf("board config = %q", content)
	}
}

func TestBoardCatalogRejectsPathLikeID(t *testing.T) {
	catalog := BuiltinBoardCatalog(resourcecache.Store{Root: t.TempDir()})
	_, err := catalog.Resolve("../robomaster-c")
	if err == nil || !strings.Contains(err.Error(), "invalid") {
		t.Fatalf("Resolve() error = %v", err)
	}
}
