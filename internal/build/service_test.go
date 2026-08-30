package build

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/x12315/rm-relay/internal/build/output"
	"github.com/x12315/rm-relay/internal/profile"
	"github.com/x12315/rm-relay/internal/project"
)

func TestServiceCreatesManifestOnlyAfterBackendSucceeds(t *testing.T) {
	outputDirectory := t.TempDir()
	plan := servicePlan(outputDirectory)
	backendError := errors.New("builder unavailable")
	service := Service{
		Backend:         fakeBackend{err: backendError},
		ProducerVersion: "0.1.0",
	}

	_, err := service.Execute(context.Background(), plan)
	if !errors.Is(err, backendError) {
		t.Fatalf("Execute() error = %v, want %v", err, backendError)
	}
	if !strings.Contains(err.Error(), "test-backend") {
		t.Fatalf("Execute() error = %v, want backend identity", err)
	}
	if _, err := os.Stat(filepath.Join(outputDirectory, output.ManifestFileName)); !os.IsNotExist(err) {
		t.Fatalf("manifest exists after failed backend: %v", err)
	}
}

func TestBackendCatalogResolvesByStableID(t *testing.T) {
	catalog, err := NewBackendCatalog(fakeBackend{})
	if err != nil {
		t.Fatal(err)
	}
	backend, err := catalog.Resolve("test-backend")
	if err != nil {
		t.Fatal(err)
	}
	if backend.ID() != "test-backend" {
		t.Fatalf("backend ID = %q", backend.ID())
	}
}

func TestBackendCatalogRejectsDuplicateIDs(t *testing.T) {
	_, err := NewBackendCatalog(fakeBackend{}, fakeBackend{})
	if err == nil || !strings.Contains(err.Error(), "multiple") {
		t.Fatalf("NewBackendCatalog() error = %v", err)
	}
}

func TestBackendCatalogReportsAvailableBackends(t *testing.T) {
	catalog, err := NewBackendCatalog(fakeBackend{id: "remote"}, fakeBackend{id: "local"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = catalog.Resolve("missing")
	if err == nil || !strings.Contains(err.Error(), "[local remote]") {
		t.Fatalf("Resolve() error = %v", err)
	}
}

func TestServiceCreatesManifestFromSuccessfulBackendOutput(t *testing.T) {
	outputDirectory := t.TempDir()
	plan := servicePlan(outputDirectory)
	service := Service{
		Backend: fakeBackend{
			imageID: "sha256:image",
			build: func() error {
				return os.WriteFile(filepath.Join(outputDirectory, "firmware.elf"), []byte("elf"), 0o644)
			},
		},
		ProducerVersion: "0.1.0",
	}

	manifest, err := service.Execute(context.Background(), plan)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if manifest.DevelopmentImageID != "sha256:image" {
		t.Fatalf("DevelopmentImageID = %q", manifest.DevelopmentImageID)
	}
}

type fakeBackend struct {
	id      string
	imageID string
	build   func() error
	err     error
}

func (backend fakeBackend) ID() string {
	if backend.id != "" {
		return backend.id
	}
	return "test-backend"
}

func (backend fakeBackend) Build(context.Context, Plan) (string, error) {
	if backend.err != nil {
		return "", backend.err
	}
	if backend.build != nil {
		if err := backend.build(); err != nil {
			return "", err
		}
	}
	return backend.imageID, nil
}

func servicePlan(outputDirectory string) Plan {
	return Plan{
		ProjectID:       "1e013e16-04a7-4fd3-9f48-bfc9178f5421",
		OutputDirectory: outputDirectory,
		Build: project.Build{
			Profile: "embedded-test",
			Outputs: []project.Output{{Role: "firmware.elf", Path: "firmware.elf"}},
		},
		Profile: profile.Loaded{
			Digest: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			Config: profile.Config{
				ID:                  "embedded-test",
				DevelopmentImage:    "mcu-dev/toolchain:test",
				RequiredOutputRoles: []string{"firmware.elf"},
			},
		},
	}
}
