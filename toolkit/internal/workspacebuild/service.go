// Package workspacebuild coordinates a resolved plan with one workspace build backend.
package workspacebuild

import (
	"context"
	"fmt"

	"github.com/x12315/rm-relay/toolkit/internal/buildoutput"
	"github.com/x12315/rm-relay/toolkit/internal/executionplan"
)

// Backend executes a workspace build and returns the immutable identity of its environment image.
type Backend interface {
	Build(context.Context, executionplan.Plan) (imageID string, err error)
}

// Service creates a Build Output only after its backend succeeds.
type Service struct {
	Backend         Backend
	ProducerVersion string
}

// Execute runs the backend and records the resulting artifacts.
func (service Service) Execute(ctx context.Context, plan executionplan.Plan) (buildoutput.Manifest, error) {
	if service.Backend == nil {
		return buildoutput.Manifest{}, fmt.Errorf("build backend is not configured")
	}
	imageID, err := service.Backend.Build(ctx, plan)
	if err != nil {
		return buildoutput.Manifest{}, fmt.Errorf("execute %s backend: %w", plan.Backend, err)
	}
	manifest, err := buildoutput.Create(plan, imageID, service.ProducerVersion)
	if err != nil {
		return buildoutput.Manifest{}, fmt.Errorf("create Build Output: %w", err)
	}
	return manifest, nil
}
