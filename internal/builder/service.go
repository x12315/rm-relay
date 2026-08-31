package builder

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/x12315/rm-relay/internal/execution/buildx"
	"github.com/x12315/rm-relay/internal/execution/docker"
)

// AddRequest contains the mTLS material Buildx needs to register a remote Builder.
type AddRequest struct {
	ID, Endpoint, CAPath, CertificatePath, KeyPath, ServerName string
}

// Manager exposes Builder resource operations to the public CLI.
type Manager interface {
	Add(context.Context, AddRequest) error
	Remove(context.Context, string) error
	SetEnvironment(string, string, string) error
	List() ([]Definition, error)
	Check(context.Context, string) error
}

// Remove deletes one logical mapping and its RM Relay-owned Buildx resource.
func (service Service) Remove(ctx context.Context, builderID string) error {
	if builderID == LocalID {
		return fmt.Errorf("built-in builder %q cannot be removed", LocalID)
	}
	if service.Buildx == nil {
		return fmt.Errorf("Buildx client is not configured")
	}
	definitions, err := service.Store.Load()
	if err != nil {
		return err
	}
	selectedIndex := -1
	for index := range definitions {
		if definitions[index].ID == builderID {
			selectedIndex = index
			break
		}
	}
	if selectedIndex < 0 {
		return fmt.Errorf("builder %q does not exist", builderID)
	}
	selected := definitions[selectedIndex]
	remaining := append([]Definition(nil), definitions[:selectedIndex]...)
	remaining = append(remaining, definitions[selectedIndex+1:]...)
	if err := service.Store.Save(remaining); err != nil {
		return fmt.Errorf("remove Builder mapping: %w", err)
	}
	if err := service.Buildx.RemoveBuilder(ctx, selected.BuildxBuilder); err != nil {
		if restoreError := service.Store.Save(definitions); restoreError != nil {
			return fmt.Errorf("remove Buildx builder: %w; restore Builder mapping: %v", err, restoreError)
		}
		return fmt.Errorf("remove Buildx builder: %w", err)
	}
	return nil
}

// Service composes persistent logical mappings with official Buildx resources.
type Service struct {
	Store  Store
	Buildx buildx.Client
	Docker docker.Client
}

// Add validates credentials, creates the Buildx resource, then commits the logical mapping.
func (service Service) Add(ctx context.Context, request AddRequest) error {
	if !IsIdentifier(request.ID) || request.ID == LocalID {
		return fmt.Errorf("builder ID %q is invalid or reserved", request.ID)
	}
	for label, path := range map[string]string{"CA": request.CAPath, "certificate": request.CertificatePath, "key": request.KeyPath} {
		info, err := os.Stat(path)
		if err != nil || !info.Mode().IsRegular() {
			return fmt.Errorf("%s path %q is not a readable regular file", label, path)
		}
	}
	definitions, err := service.Store.Load()
	if err != nil {
		return err
	}
	for _, definition := range definitions {
		if definition.ID == request.ID {
			return fmt.Errorf("builder %q already exists", request.ID)
		}
	}
	if service.Buildx == nil {
		return fmt.Errorf("Buildx client is not configured")
	}
	buildxName := "rm-relay-" + request.ID
	if err := service.Buildx.CreateRemote(ctx, buildx.CreateRemoteRequest{Name: buildxName, Endpoint: request.Endpoint, CAPath: request.CAPath, CertificatePath: request.CertificatePath, KeyPath: request.KeyPath, ServerName: request.ServerName}); err != nil {
		return err
	}
	definition := Definition{ID: request.ID, Kind: KindRemoteBuildKit, BuildxBuilder: buildxName, Environments: map[string]string{}}
	if err := service.Store.Save(append(definitions, definition)); err != nil {
		rollbackError := service.Buildx.RemoveBuilder(ctx, buildxName)
		if rollbackError != nil {
			return fmt.Errorf("save Builder mapping: %w; rollback Buildx builder: %v", err, rollbackError)
		}
		return fmt.Errorf("save Builder mapping: %w", err)
	}
	return nil
}

// SetEnvironment pins one Profile environment ID to an immutable registry reference.
func (service Service) SetEnvironment(builderID, environmentID, reference string) error {
	if !IsIdentifier(environmentID) {
		return fmt.Errorf("environment ID %q is invalid", environmentID)
	}
	if !IsDigestReference(reference) {
		return fmt.Errorf("environment reference must use image@sha256:<digest>")
	}
	definitions, err := service.Store.Load()
	if err != nil {
		return err
	}
	found := false
	for index := range definitions {
		if definitions[index].ID != builderID {
			continue
		}
		if definitions[index].Environments == nil {
			definitions[index].Environments = map[string]string{}
		}
		definitions[index].Environments[environmentID] = reference
		found = true
	}
	if !found {
		return fmt.Errorf("builder %q does not exist", builderID)
	}
	return service.Store.Save(definitions)
}

// List returns the built-in local Builder and sorted persistent definitions.
func (service Service) List() ([]Definition, error) {
	definitions, err := service.Store.Load()
	if err != nil {
		return nil, err
	}
	definitions = append(definitions, Definition{ID: LocalID, Kind: KindLocalContainer})
	sort.Slice(definitions, func(i, j int) bool { return definitions[i].ID < definitions[j].ID })
	return definitions, nil
}

// Check performs a real BuildKit solve for remote Builders.
func (service Service) Check(ctx context.Context, builderID string) error {
	definitions, err := service.List()
	if err != nil {
		return err
	}
	var selected *Definition
	for index := range definitions {
		if definitions[index].ID == builderID {
			selected = &definitions[index]
			break
		}
	}
	if selected == nil {
		return fmt.Errorf("builder %q does not exist", builderID)
	}
	if selected.Kind == KindLocalContainer {
		if service.Docker == nil {
			return fmt.Errorf("Docker client is not configured")
		}
		return service.Docker.CheckEngine(ctx)
	}
	if service.Buildx == nil {
		return fmt.Errorf("Buildx client is not configured")
	}
	if err := service.Buildx.InspectBuilder(ctx, selected.BuildxBuilder); err != nil {
		return err
	}
	root, err := os.MkdirTemp("", "rm-relay-builder-check-*")
	if err != nil {
		return fmt.Errorf("create Builder check workspace: %w", err)
	}
	defer os.RemoveAll(root)
	contextDirectory := filepath.Join(root, "context")
	outputDirectory := filepath.Join(root, "output")
	if err := os.MkdirAll(contextDirectory, 0o700); err != nil {
		return err
	}
	if err := os.MkdirAll(outputDirectory, 0o700); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(contextDirectory, "probe"), []byte("rm-relay\n"), 0o600); err != nil {
		return err
	}
	dockerfile := strings.NewReader("FROM scratch\nCOPY probe /probe\nFROM scratch\nCOPY --from=0 /probe /probe\n")
	if err := service.Buildx.Build(ctx, buildx.BuildRequest{Builder: selected.BuildxBuilder, ContextDirectory: contextDirectory, OutputDirectory: outputDirectory, Dockerfile: dockerfile}); err != nil {
		return err
	}
	content, err := os.ReadFile(filepath.Join(outputDirectory, "probe"))
	if err != nil || string(content) != "rm-relay\n" {
		return fmt.Errorf("Builder check output is incomplete")
	}
	return nil
}
