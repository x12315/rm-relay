package builder

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/x12315/rm-relay/internal/environment"
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
	RegisterEnvironment(context.Context, string, string, string) error
	CheckEnvironment(context.Context, string, string) error
	List() ([]Definition, error)
	Prepare(context.Context, string) error
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
	Store               Store
	Buildx              buildx.Client
	Docker              docker.Client
	EnvironmentVerifier environment.Verifier
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

// RegisterEnvironment verifies and pins one environment image for a logical Builder.
func (service Service) RegisterEnvironment(ctx context.Context, builderID, environmentID, reference string) error {
	if !environment.IsIdentifier(environmentID) {
		return fmt.Errorf("environment ID %q is invalid", environmentID)
	}
	if _, err := environment.ParseDigestReference(reference); err != nil {
		return err
	}
	if service.EnvironmentVerifier == nil {
		return fmt.Errorf("environment verifier is not configured")
	}
	definition, err := service.resolve(builderID)
	if err != nil {
		return err
	}
	if err := service.Prepare(ctx, builderID); err != nil {
		return err
	}
	identity, err := service.EnvironmentVerifier.Verify(ctx, definition.BuildxBuilder, reference)
	if err != nil {
		return err
	}
	if identity.ID != environmentID {
		return fmt.Errorf("environment image identity %q does not match requested ID %q", identity.ID, environmentID)
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
	if !found && builderID == LocalID {
		local := canonicalLocalDefinition()
		local.Environments[environmentID] = reference
		definitions = append(definitions, local)
		found = true
	}
	if !found {
		return fmt.Errorf("builder %q does not exist", builderID)
	}
	return service.Store.Save(definitions)
}

// CheckEnvironment re-verifies the image currently pinned for one logical Builder.
func (service Service) CheckEnvironment(ctx context.Context, builderID, environmentID string) error {
	definition, err := service.resolve(builderID)
	if err != nil {
		return err
	}
	reference, err := definition.EnvironmentReference(environmentID)
	if err != nil {
		return err
	}
	if service.EnvironmentVerifier == nil {
		return fmt.Errorf("environment verifier is not configured")
	}
	if err := service.Prepare(ctx, builderID); err != nil {
		return err
	}
	identity, err := service.EnvironmentVerifier.Verify(ctx, definition.BuildxBuilder, reference)
	if err != nil {
		return err
	}
	if identity.ID != environmentID {
		return fmt.Errorf("environment image identity %q does not match registered ID %q", identity.ID, environmentID)
	}
	return nil
}

// List returns the built-in local Builder and sorted persistent definitions.
func (service Service) List() ([]Definition, error) {
	definitions, err := service.Store.Load()
	if err != nil {
		return nil, err
	}
	catalog, err := NewCatalog(definitions...)
	if err != nil {
		return nil, err
	}
	listed := make([]Definition, 0, len(catalog.definitions))
	for _, definition := range catalog.definitions {
		listed = append(listed, definition)
	}
	sort.Slice(listed, func(i, j int) bool { return listed[i].ID < listed[j].ID })
	return listed, nil
}

func (service Service) resolve(builderID string) (Definition, error) {
	definitions, err := service.Store.Load()
	if err != nil {
		return Definition{}, err
	}
	catalog, err := NewCatalog(definitions...)
	if err != nil {
		return Definition{}, err
	}
	return catalog.Resolve(builderID)
}

// Prepare makes one registered Builder ready without changing the user's active Buildx selection.
func (service Service) Prepare(ctx context.Context, builderID string) error {
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
	if service.Buildx == nil {
		return fmt.Errorf("Buildx client is not configured")
	}
	if selected.Kind == KindRemoteBuildKit {
		return service.Buildx.InspectBuilder(ctx, selected.BuildxBuilder)
	}
	if service.Docker == nil {
		return fmt.Errorf("Docker client is not configured")
	}
	if err := service.Docker.CheckEngine(ctx); err != nil {
		return err
	}
	registered, err := service.Buildx.ListBuilders(ctx)
	if err != nil {
		return err
	}
	for _, resource := range registered {
		if resource.Name != LocalBuildxBuilder {
			continue
		}
		if resource.Driver != "docker-container" {
			return fmt.Errorf("Buildx resource %q uses driver %q; RM Relay requires docker-container", LocalBuildxBuilder, resource.Driver)
		}
		return service.Buildx.InspectBuilder(ctx, LocalBuildxBuilder)
	}
	return service.Buildx.CreateLocal(ctx, buildx.CreateLocalRequest{Name: LocalBuildxBuilder, Image: LocalBuildKitImage})
}

// Check prepares a Builder and proves it can execute a real BuildKit solve.
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
	if err := service.Prepare(ctx, builderID); err != nil {
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
