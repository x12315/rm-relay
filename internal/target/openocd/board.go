package openocd

import (
	"embed"
	"fmt"
	"io/fs"
	"path"

	"github.com/x12315/rm-relay/internal/execution/resourcecache"
)

//go:embed board/*.cfg
var builtinBoardFiles embed.FS

// BoardCatalog resolves semantic board IDs to OpenOCD configuration files.
type BoardCatalog struct {
	Files fs.FS
	Root  string
	Store resourcecache.Store
}

// BuiltinBoardCatalog returns the OpenOCD boards shipped with rm-relay.
func BuiltinBoardCatalog(store resourcecache.Store) BoardCatalog {
	return BoardCatalog{Files: builtinBoardFiles, Root: "board", Store: store}
}

// Resolve returns a stable local path for the requested board definition.
func (catalog BoardCatalog) Resolve(boardID string) (string, error) {
	if !safeBoardID(boardID) {
		return "", fmt.Errorf("OpenOCD board ID %q is invalid", boardID)
	}
	if catalog.Files == nil {
		return "", fmt.Errorf("OpenOCD board catalog filesystem is not configured")
	}
	content, err := fs.ReadFile(catalog.Files, path.Join(catalog.Root, boardID+".cfg"))
	if err != nil {
		return "", fmt.Errorf("read OpenOCD board %q: %w", boardID, err)
	}
	configPath, err := catalog.Store.Materialize("target/openocd/board", boardID+".cfg", content)
	if err != nil {
		return "", fmt.Errorf("materialize OpenOCD board %q: %w", boardID, err)
	}
	return configPath, nil
}

func safeBoardID(value string) bool {
	if value == "" {
		return false
	}
	for index, character := range value {
		if (character >= 'a' && character <= 'z') ||
			(character >= 'A' && character <= 'Z') ||
			(character >= '0' && character <= '9') ||
			(index > 0 && (character == '-' || character == '_' || character == '.')) {
			continue
		}
		return false
	}
	return true
}
