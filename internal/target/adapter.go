// Package target defines the handoff from a verified Build Output to a development target.
package target

import (
	"context"
	"fmt"
	"sort"

	"github.com/x12315/rm-relay/internal/build/output"
	"github.com/x12315/rm-relay/internal/profile"
)

// FlashRequest identifies a verified artifact and one Profile target capability.
type FlashRequest struct {
	BuildOutput output.Verified
	Profile     profile.Loaded
	TargetName  string
	DryRun      bool
}

// FlashResult records the native command and whether it was executed.
type FlashResult struct {
	Command  []string
	Executed bool
}

// FlashAdapter sends a verified Build Output to one kind of flash target.
type FlashAdapter interface {
	ID() string
	Flash(context.Context, FlashRequest) (FlashResult, error)
}

// FlashAdapterCatalog resolves Profile adapter IDs without exposing implementations to the CLI.
type FlashAdapterCatalog struct {
	byID map[string]FlashAdapter
}

// NewFlashAdapterCatalog validates and indexes the supplied flash adapters.
func NewFlashAdapterCatalog(adapters ...FlashAdapter) (FlashAdapterCatalog, error) {
	byID := make(map[string]FlashAdapter, len(adapters))
	for _, adapter := range adapters {
		if adapter == nil {
			return FlashAdapterCatalog{}, fmt.Errorf("flash adapter must not be nil")
		}
		adapterID := adapter.ID()
		if adapterID == "" {
			return FlashAdapterCatalog{}, fmt.Errorf("flash adapter ID must not be empty")
		}
		if _, exists := byID[adapterID]; exists {
			return FlashAdapterCatalog{}, fmt.Errorf("multiple flash adapters use ID %q", adapterID)
		}
		byID[adapterID] = adapter
	}
	return FlashAdapterCatalog{byID: byID}, nil
}

// Resolve returns the flash adapter registered for adapterID.
func (catalog FlashAdapterCatalog) Resolve(adapterID string) (FlashAdapter, error) {
	adapter, exists := catalog.byID[adapterID]
	if exists {
		return adapter, nil
	}
	available := make([]string, 0, len(catalog.byID))
	for registeredID := range catalog.byID {
		available = append(available, registeredID)
	}
	sort.Strings(available)
	return nil, fmt.Errorf("unsupported flash adapter %q; available adapters: %v", adapterID, available)
}
