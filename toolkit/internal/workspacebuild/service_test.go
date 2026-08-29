package workspacebuild

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/x12315/rm-relay/toolkit/internal/buildoutput"
	"github.com/x12315/rm-relay/toolkit/internal/executionplan"
	"github.com/x12315/rm-relay/toolkit/internal/profile"
	"github.com/x12315/rm-relay/toolkit/internal/project"
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
	if _, err := os.Stat(filepath.Join(outputDirectory, buildoutput.ManifestFileName)); !os.IsNotExist(err) {
		t.Fatalf("manifest exists after failed backend: %v", err)
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
	imageID string
	build   func() error
	err     error
}

func (backend fakeBackend) Build(context.Context, executionplan.Plan) (string, error) {
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

func servicePlan(outputDirectory string) executionplan.Plan {
	return executionplan.Plan{
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
