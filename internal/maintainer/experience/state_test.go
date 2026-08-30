package experience

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestStateRoundTrip(t *testing.T) {
	layout, err := ResolveLayout(t.TempDir(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(layout.Root, 0o755); err != nil {
		t.Fatal(err)
	}
	want := validState(layout)

	if err := writeState(layout, want); err != nil {
		t.Fatal(err)
	}
	actual, err := readState(layout)
	if err != nil {
		t.Fatal(err)
	}
	if actual != want {
		t.Fatalf("state = %#v, want %#v", actual, want)
	}
}

func TestReadStateRejectsSymlink(t *testing.T) {
	layout, err := ResolveLayout(t.TempDir(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(layout.Root, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "state.json")
	if err := os.WriteFile(target, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, layout.StatePath); err != nil {
		t.Skipf("create state symlink: %v", err)
	}

	_, err = readState(layout)
	if err == nil || !strings.Contains(err.Error(), "regular file") {
		t.Fatalf("readState() error = %v", err)
	}
}

func TestValidateStateRejectsWrongOwnership(t *testing.T) {
	layout, err := ResolveLayout(t.TempDir(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	state := validState(layout)
	state.Marker = "not-rm-relay"

	err = validateState(layout, state)
	if err == nil || !strings.Contains(err.Error(), "marker") {
		t.Fatalf("validateState() error = %v", err)
	}

	state = validState(layout)
	state.RepositoryKey = "different"
	err = validateState(layout, state)
	if err == nil || !strings.Contains(err.Error(), "repository key") {
		t.Fatalf("validateState() error = %v", err)
	}
}

func validState(layout Layout) State {
	return State{
		SchemaVersion:    StateSchemaVersion,
		Marker:           managedStateMarker,
		RepositoryRoot:   layout.RepositoryRoot,
		RepositoryKey:    layout.RepositoryKey,
		Revision:         "0123456789abcdef",
		CLIVersion:       "0.0.0-SNAPSHOT-test",
		CLISHA256:        strings.Repeat("a", 64),
		ImageReference:   developmentImageReference,
		ImageID:          "sha256:candidate",
		PreviousImageID:  "sha256:previous",
		TemplateRevision: "abcdef0123456789",
		CreatedAt:        time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC),
	}
}
