package candidate

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/x12315/rm-relay/internal/builder"
)

const (
	// StateSchemaVersion is the candidate environment state format understood by this CLI.
	StateSchemaVersion = 2
	managedStateMarker = "rm-relay-candidate-experience"

	developmentImageReference = "mcu-dev/toolchain:local"
	sha256HexLength           = 64
)

// State records the identities required to enter or safely clean one candidate environment.
type State struct {
	SchemaVersion        int       `json:"schema_version"`
	Marker               string    `json:"marker"`
	RepositoryRoot       string    `json:"repository_root"`
	RepositoryKey        string    `json:"repository_key"`
	Revision             string    `json:"revision"`
	CLIVersion           string    `json:"cli_version"`
	CLISHA256            string    `json:"cli_sha256"`
	ImageReference       string    `json:"image_reference"`
	ImageID              string    `json:"image_id"`
	EnvironmentReference string    `json:"environment_reference"`
	PreviousImageID      string    `json:"previous_image_id,omitempty"`
	TemplateRevision     string    `json:"template_revision"`
	CreatedAt            time.Time `json:"created_at"`
}

func writeState(layout Layout, state State) error {
	if err := validateState(layout, state); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(layout.Root, ".state-*.json")
	if err != nil {
		return fmt.Errorf("create temporary candidate state: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o644); err != nil {
		temporary.Close()
		return fmt.Errorf("set candidate state permissions: %w", err)
	}
	encoder := json.NewEncoder(temporary)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(state); err != nil {
		temporary.Close()
		return fmt.Errorf("encode candidate state: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return fmt.Errorf("sync candidate state: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close candidate state: %w", err)
	}
	if err := os.Rename(temporaryPath, layout.StatePath); err != nil {
		return fmt.Errorf("publish candidate state: %w", err)
	}
	return nil
}

func readState(layout Layout) (State, error) {
	info, err := os.Lstat(layout.StatePath)
	if err != nil {
		return State{}, fmt.Errorf("inspect candidate state: %w", err)
	}
	if !info.Mode().IsRegular() {
		return State{}, fmt.Errorf("candidate state must be a regular file: %s", layout.StatePath)
	}
	file, err := os.Open(layout.StatePath)
	if err != nil {
		return State{}, fmt.Errorf("open candidate state: %w", err)
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var state State
	if err := decoder.Decode(&state); err != nil {
		return State{}, fmt.Errorf("decode candidate state: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return State{}, fmt.Errorf("candidate state contains trailing data")
	}
	if err := validateState(layout, state); err != nil {
		return State{}, err
	}
	return state, nil
}

func validateState(layout Layout, state State) error {
	if state.SchemaVersion != StateSchemaVersion {
		return fmt.Errorf("candidate state schema version %d is unsupported", state.SchemaVersion)
	}
	if state.Marker != managedStateMarker {
		return fmt.Errorf("candidate state marker is invalid")
	}
	if state.RepositoryRoot != layout.RepositoryRoot {
		return fmt.Errorf("candidate state repository root does not match %q", layout.RepositoryRoot)
	}
	if state.RepositoryKey != layout.RepositoryKey {
		return fmt.Errorf("candidate state repository key does not match %q", layout.RepositoryKey)
	}
	if state.Revision == "" || state.CLIVersion == "" || len(state.CLISHA256) != sha256HexLength {
		return fmt.Errorf("candidate state CLI identity is incomplete")
	}
	if state.ImageReference != developmentImageReference || state.ImageID == "" {
		return fmt.Errorf("candidate state image identity is invalid")
	}
	if !builder.IsDigestReference(state.EnvironmentReference) {
		return fmt.Errorf("candidate state environment reference is invalid")
	}
	if state.TemplateRevision == "" || state.CreatedAt.IsZero() {
		return fmt.Errorf("candidate state template identity or creation time is incomplete")
	}
	return nil
}
