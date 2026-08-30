package target

import (
	"context"
	"strings"
	"testing"
)

func TestFlashAdapterCatalogResolvesByStableID(t *testing.T) {
	catalog, err := NewFlashAdapterCatalog(testFlashAdapter{id: "openocd"})
	if err != nil {
		t.Fatal(err)
	}
	adapter, err := catalog.Resolve("openocd")
	if err != nil {
		t.Fatal(err)
	}
	if adapter.ID() != "openocd" {
		t.Fatalf("adapter ID = %q", adapter.ID())
	}
}

func TestFlashAdapterCatalogRejectsDuplicateIDs(t *testing.T) {
	_, err := NewFlashAdapterCatalog(testFlashAdapter{id: "openocd"}, testFlashAdapter{id: "openocd"})
	if err == nil || !strings.Contains(err.Error(), "multiple") {
		t.Fatalf("NewFlashAdapterCatalog() error = %v", err)
	}
}

func TestFlashAdapterCatalogReportsAvailableAdapters(t *testing.T) {
	catalog, err := NewFlashAdapterCatalog(testFlashAdapter{id: "serial"}, testFlashAdapter{id: "openocd"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = catalog.Resolve("missing")
	if err == nil || !strings.Contains(err.Error(), "[openocd serial]") {
		t.Fatalf("Resolve() error = %v", err)
	}
}

type testFlashAdapter struct {
	id string
}

func (adapter testFlashAdapter) ID() string {
	return adapter.id
}

func (testFlashAdapter) Flash(context.Context, FlashRequest) (FlashResult, error) {
	return FlashResult{}, nil
}
