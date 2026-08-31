package build

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/x12315/rm-relay/internal/build/output"
	"github.com/x12315/rm-relay/internal/builder"
	"github.com/x12315/rm-relay/internal/profile"
	"github.com/x12315/rm-relay/internal/project"
)

func TestServiceCreatesManifestOnlyAfterBackendSucceeds(t *testing.T) {
	outputDirectory := t.TempDir()
	plan := servicePlan(outputDirectory)
	backendError := errors.New("builder unavailable")
	service := Service{
		Backend:         fakeBackend{err: backendError},
		Builder:         testBuilder(),
		ProducerVersion: "0.1.0",
	}

	_, err := service.Execute(context.Background(), plan)
	if !errors.Is(err, backendError) {
		t.Fatalf("Execute() error = %v, want %v", err, backendError)
	}
	if !strings.Contains(err.Error(), "local-container") {
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
	backend, err := catalog.Resolve(string(builder.KindLocalContainer))
	if err != nil {
		t.Fatal(err)
	}
	if backend.Kind() != builder.KindLocalContainer {
		t.Fatalf("backend kind = %q", backend.Kind())
	}
}

func TestBackendCatalogRejectsDuplicateIDs(t *testing.T) {
	_, err := NewBackendCatalog(fakeBackend{}, fakeBackend{})
	if err == nil || !strings.Contains(err.Error(), "multiple") {
		t.Fatalf("NewBackendCatalog() error = %v", err)
	}
}

func TestBackendCatalogReportsAvailableBackends(t *testing.T) {
	catalog, err := NewBackendCatalog(fakeBackend{kind: builder.KindRemoteBuildKit}, fakeBackend{kind: builder.KindLocalContainer})
	if err != nil {
		t.Fatal(err)
	}
	_, err = catalog.Resolve("missing")
	if err == nil || !strings.Contains(err.Error(), "[local-container remote-buildkit]") {
		t.Fatalf("Resolve() error = %v", err)
	}
}

func TestServiceCreatesManifestFromSuccessfulBackendOutput(t *testing.T) {
	outputDirectory := t.TempDir()
	plan := servicePlan(outputDirectory)
	service := Service{
		Backend: fakeBackend{
			evidence: testEvidence(),
			build: func() error {
				return os.WriteFile(filepath.Join(outputDirectory, "firmware.elf"), []byte("elf"), 0o644)
			},
		},
		Builder:         testBuilder(),
		ProducerVersion: "0.1.0",
	}

	manifest, err := service.Execute(context.Background(), plan)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if manifest.Environment.Digest != testEvidence().EnvironmentDigest {
		t.Fatalf("environment digest = %q", manifest.Environment.Digest)
	}
}

type fakeBackend struct {
	kind     builder.Kind
	evidence ExecutionEvidence
	build    func() error
	err      error
}

func (backend fakeBackend) Kind() builder.Kind {
	if backend.kind != "" {
		return backend.kind
	}
	return builder.KindLocalContainer
}

func (backend fakeBackend) Build(context.Context, Plan, builder.Definition) (ExecutionEvidence, error) {
	if backend.err != nil {
		return ExecutionEvidence{}, backend.err
	}
	if backend.build != nil {
		if err := backend.build(); err != nil {
			return ExecutionEvidence{}, err
		}
	}
	return backend.evidence, nil
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
				Environment:         profile.Environment{ID: "embedded-development", LocalReference: "mcu-dev/toolchain:test"},
				RequiredOutputRoles: []string{"firmware.elf"},
			},
		},
	}
}

func testBuilder() builder.Definition {
	return builder.Definition{ID: builder.LocalID, Kind: builder.KindLocalContainer}
}

func testEvidence() ExecutionEvidence {
	return ExecutionEvidence{
		BuilderID: builder.LocalID, BuilderKind: string(builder.KindLocalContainer),
		EnvironmentID: "embedded-development", EnvironmentReference: "mcu-dev/toolchain:test",
		EnvironmentDigest: "sha256:" + strings.Repeat("b", 64),
	}
}
