// Package build coordinates resolved plans with workspace build backends.
package build

import (
	"context"
	"fmt"
	"sort"

	"github.com/x12315/rm-relay/internal/build/output"
)

// Backend executes a workspace build and returns the immutable identity of its environment image.
type Backend interface {
	ID() string
	Build(context.Context, Plan) (imageID string, err error)
}

// BackendCatalog resolves build backends by stable ID without coupling the CLI to implementations.
type BackendCatalog struct {
	byID map[string]Backend
}

// NewBackendCatalog validates and indexes the supplied build backends.
func NewBackendCatalog(backends ...Backend) (BackendCatalog, error) {
	byID := make(map[string]Backend, len(backends))
	for _, backend := range backends {
		if backend == nil {
			return BackendCatalog{}, fmt.Errorf("build backend must not be nil")
		}
		backendID := backend.ID()
		if backendID == "" {
			return BackendCatalog{}, fmt.Errorf("build backend ID must not be empty")
		}
		if _, exists := byID[backendID]; exists {
			return BackendCatalog{}, fmt.Errorf("multiple build backends use ID %q", backendID)
		}
		byID[backendID] = backend
	}
	return BackendCatalog{byID: byID}, nil
}

// Resolve returns the backend registered for backendID.
func (catalog BackendCatalog) Resolve(backendID string) (Backend, error) {
	backend, exists := catalog.byID[backendID]
	if exists {
		return backend, nil
	}
	available := make([]string, 0, len(catalog.byID))
	for registeredID := range catalog.byID {
		available = append(available, registeredID)
	}
	sort.Strings(available)
	return nil, fmt.Errorf("unsupported build backend %q; available backends: %v", backendID, available)
}

// Service creates a Build Output only after its backend succeeds.
type Service struct {
	Backend         Backend
	ProducerVersion string
}

// Execute runs the backend and records the resulting artifacts.
func (service Service) Execute(ctx context.Context, plan Plan) (output.Manifest, error) {
	if service.Backend == nil {
		return output.Manifest{}, fmt.Errorf("build backend is not configured")
	}
	backendID := service.Backend.ID()
	if backendID == "" {
		return output.Manifest{}, fmt.Errorf("build backend identity is empty")
	}
	imageID, err := service.Backend.Build(ctx, plan)
	if err != nil {
		return output.Manifest{}, fmt.Errorf("execute %s backend: %w", backendID, err)
	}
	manifest, err := output.Create(output.CreateRequest{
		OutputDirectory: plan.OutputDirectory,
		ProjectID:       plan.ProjectID,
		Profile:         plan.Profile,
		DeclaredOutputs: plan.Build.Outputs,
		ImageID:         imageID,
		ProducerVersion: service.ProducerVersion,
	})
	if err != nil {
		return output.Manifest{}, fmt.Errorf("create Build Output: %w", err)
	}
	return manifest, nil
}
