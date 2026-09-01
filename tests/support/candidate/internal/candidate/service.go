package candidate

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/x12315/rm-relay/internal/builder"
	"github.com/x12315/rm-relay/internal/environment"
	"github.com/x12315/rm-relay/internal/execution/command"
)

// BinaryBuilder produces the current-platform candidate CLI outside the repository.
type BinaryBuilder interface {
	BuildHostBinary(context.Context, string) (Binary, error)
}

// Binary identifies one candidate CLI file.
type Binary struct{ Path, Version, SHA256 string }

// Prepared describes the candidate environment a maintainer can enter.
type Prepared struct {
	Root                 string
	Revision             string
	CLIVersion           string
	BuilderID            string
	EnvironmentReference string
	TemplateURL          string
}

// Service manages one repository's disposable candidate experience environment.
type Service struct {
	RepositoryRoot       string
	UserCacheRoot        string
	Builder              builder.Definition
	EnvironmentID        string
	EnvironmentReference string
	Runner               command.Runner
	BinaryBuilder        BinaryBuilder
	Now                  func() time.Time
	Shell                string
	Stdin                io.Reader
	Stdout               io.Writer
	Stderr               io.Writer
}

// Prepare creates a candidate CLI, isolated resource mapping, template origin and empty workspace outside the repository.
func (service Service) Prepare(ctx context.Context) (Prepared, error) {
	layout, err := service.validatedLayout()
	if err != nil {
		return Prepared{}, err
	}
	if service.BinaryBuilder == nil {
		return Prepared{}, fmt.Errorf("candidate CLI builder is not configured")
	}
	if service.Runner == nil {
		return Prepared{}, fmt.Errorf("process runner is not configured")
	}
	selectedBuilder, err := service.isolatedBuilderDefinition()
	if err != nil {
		return Prepared{}, err
	}
	if _, err := os.Lstat(layout.Root); err == nil {
		return Prepared{}, fmt.Errorf("candidate experience already exists at %s", layout.Root)
	} else if !os.IsNotExist(err) {
		return Prepared{}, fmt.Errorf("inspect candidate experience: %w", err)
	}
	if err := service.requireCleanRepository(ctx, layout.RepositoryRoot); err != nil {
		return Prepared{}, err
	}
	revision, err := service.gitRevision(ctx, layout.RepositoryRoot)
	if err != nil {
		return Prepared{}, err
	}
	if err := os.MkdirAll(filepath.Dir(layout.Root), 0o755); err != nil {
		return Prepared{}, fmt.Errorf("create candidate experience parent: %w", err)
	}
	preparingRoot, err := os.MkdirTemp(filepath.Dir(layout.Root), "."+layout.RepositoryKey+".preparing-")
	if err != nil {
		return Prepared{}, fmt.Errorf("create candidate preparation directory: %w", err)
	}
	preparingLayout := layoutWithRoot(layout, preparingRoot)
	published := false
	defer func() {
		if !published {
			_ = os.RemoveAll(preparingRoot)
		}
	}()

	for _, directory := range []string{preparingLayout.BinaryDirectory, preparingLayout.ConfigDirectory, preparingLayout.Workspace, preparingLayout.Logs} {
		if err := os.MkdirAll(directory, 0o755); err != nil {
			return Prepared{}, fmt.Errorf("create candidate directory: %w", err)
		}
	}
	binary, err := service.BinaryBuilder.BuildHostBinary(ctx, preparingLayout.BinaryPath)
	if err != nil {
		return Prepared{}, fmt.Errorf("build candidate CLI: %w", err)
	}
	templateRevision, err := service.createTemplateOrigin(ctx, preparingLayout, revision)
	if err != nil {
		return Prepared{}, err
	}
	if err := (builder.Store{Directory: filepath.Join(preparingLayout.ConfigDirectory, "rm-relay")}).Save([]builder.Definition{selectedBuilder}); err != nil {
		return Prepared{}, fmt.Errorf("configure candidate Builder: %w", err)
	}

	now := time.Now
	if service.Now != nil {
		now = service.Now
	}
	state := State{
		SchemaVersion:        StateSchemaVersion,
		Marker:               managedStateMarker,
		RepositoryRoot:       layout.RepositoryRoot,
		RepositoryKey:        layout.RepositoryKey,
		Revision:             revision,
		CLIVersion:           binary.Version,
		CLISHA256:            binary.SHA256,
		BuilderID:            selectedBuilder.ID,
		BuilderKind:          selectedBuilder.Kind,
		BuildxBuilder:        selectedBuilder.BuildxBuilder,
		EnvironmentID:        service.EnvironmentID,
		EnvironmentReference: service.EnvironmentReference,
		TemplateRevision:     templateRevision,
		CreatedAt:            now().UTC(),
	}
	if err := writeState(preparingLayout, state); err != nil {
		return Prepared{}, err
	}
	if err := os.Rename(preparingRoot, layout.Root); err != nil {
		return Prepared{}, fmt.Errorf("publish candidate experience: %w", err)
	}
	published = true
	return Prepared{
		Root:                 layout.Root,
		Revision:             revision,
		CLIVersion:           binary.Version,
		BuilderID:            selectedBuilder.ID,
		EnvironmentReference: service.EnvironmentReference,
		TemplateURL:          templateURL(layout.TemplateOrigin),
	}, nil
}

// Enter validates candidate identities and opens an interactive shell in its empty workspace.
func (service Service) Enter(ctx context.Context) error {
	layout, state, err := service.loadCandidate()
	if err != nil {
		return err
	}
	if service.Runner == nil {
		return fmt.Errorf("process runner is not configured")
	}
	if err := service.requireCleanRepository(ctx, layout.RepositoryRoot); err != nil {
		return err
	}
	revision, err := service.gitRevision(ctx, layout.RepositoryRoot)
	if err != nil {
		return err
	}
	if revision != state.Revision {
		return fmt.Errorf("candidate revision is %s, current repository revision is %s", state.Revision, revision)
	}
	digest, err := fileSHA256(layout.BinaryPath)
	if err != nil {
		return err
	}
	if digest != state.CLISHA256 {
		return fmt.Errorf("candidate CLI identity changed: got %s, want %s", digest, state.CLISHA256)
	}
	versionResult, err := service.Runner.Run(ctx, command.Request{Name: layout.BinaryPath, Arguments: []string{"--version"}})
	if err != nil {
		return candidateProcessFailure("read candidate CLI version", versionResult, err)
	}
	if strings.TrimSpace(versionResult.Stdout) != "rm-relay version "+state.CLIVersion {
		return fmt.Errorf("candidate CLI version output does not match %q", state.CLIVersion)
	}
	definitions, err := (builder.Store{Directory: filepath.Join(layout.ConfigDirectory, "rm-relay")}).Load()
	if err != nil {
		return fmt.Errorf("load candidate Builder catalog: %w", err)
	}
	if len(definitions) != 1 || definitions[0].ID != state.BuilderID {
		return fmt.Errorf("candidate Builder catalog identity changed")
	}
	catalog, err := builder.NewCatalog(definitions...)
	if err != nil {
		return fmt.Errorf("resolve candidate Builder catalog: %w", err)
	}
	selectedBuilder, err := catalog.Resolve(state.BuilderID)
	if err != nil {
		return err
	}
	if selectedBuilder.Kind != state.BuilderKind || selectedBuilder.BuildxBuilder != state.BuildxBuilder {
		return fmt.Errorf("candidate Builder identity changed")
	}
	environmentReference, err := selectedBuilder.EnvironmentReference(state.EnvironmentID)
	if err != nil {
		return err
	}
	if environmentReference != state.EnvironmentReference {
		return fmt.Errorf("candidate environment identity changed: got %s, want %s", environmentReference, state.EnvironmentReference)
	}
	templateRevision, err := service.bareRepositoryRevision(ctx, layout.TemplateOrigin)
	if err != nil {
		return err
	}
	if templateRevision != state.TemplateRevision {
		return fmt.Errorf("candidate template identity changed: got %s, want %s", templateRevision, state.TemplateRevision)
	}
	if service.Shell == "" {
		return fmt.Errorf("candidate shell is not configured")
	}
	if service.Stdout != nil {
		fmt.Fprintf(service.Stdout, "Candidate revision: %s\nCandidate CLI: %s\nBuilder: %s\nEnvironment: %s\nTemplate: %s\nClone with: git clone \"$RM_RELAY_TEMPLATE_URL\" project\n", state.Revision, state.CLIVersion, state.BuilderID, state.EnvironmentReference, templateURL(layout.TemplateOrigin))
	}
	_, err = service.Runner.Run(ctx, command.Request{
		Name:      service.Shell,
		Directory: layout.Workspace,
		Environment: map[string]string{
			"PATH":                  layout.BinaryDirectory + string(os.PathListSeparator) + os.Getenv("PATH"),
			"RM_RELAY_CONFIG_DIR":   layout.ConfigDirectory,
			"RM_RELAY_TEMPLATE_URL": templateURL(layout.TemplateOrigin),
		},
		Stdin:       service.Stdin,
		Stdout:      service.Stdout,
		Stderr:      service.Stderr,
		Interactive: true,
	})
	if err != nil {
		return fmt.Errorf("run candidate shell: %w", err)
	}
	return nil
}

// Clean removes only the files owned by the candidate environment.
func (service Service) Clean(_ context.Context) error {
	layout, _, err := service.loadCandidate()
	if err != nil {
		return err
	}
	if err := os.RemoveAll(layout.Root); err != nil {
		return fmt.Errorf("remove candidate experience: %w", err)
	}
	return nil
}

func (service Service) validatedLayout() (Layout, error) {
	return ResolveLayout(service.RepositoryRoot, service.UserCacheRoot)
}

func (service Service) isolatedBuilderDefinition() (builder.Definition, error) {
	if err := builder.ValidateDefinition(service.Builder); err != nil {
		return builder.Definition{}, fmt.Errorf("candidate Builder is invalid: %w", err)
	}
	if !environment.IsIdentifier(service.EnvironmentID) {
		return builder.Definition{}, fmt.Errorf("RM_RELAY_CANDIDATE_ENVIRONMENT_ID is invalid")
	}
	if _, err := environment.ParseDigestReference(service.EnvironmentReference); err != nil {
		return builder.Definition{}, fmt.Errorf("RM_RELAY_CANDIDATE_ENVIRONMENT must use image@sha256:<digest>")
	}
	environments := make(map[string]string, len(service.Builder.Environments)+1)
	for environmentID, reference := range service.Builder.Environments {
		environments[environmentID] = reference
	}
	environments[service.EnvironmentID] = service.EnvironmentReference
	definition := service.Builder
	definition.Environments = environments
	if err := builder.ValidateDefinition(definition); err != nil {
		return builder.Definition{}, fmt.Errorf("candidate Builder mapping is invalid: %w", err)
	}
	return definition, nil
}

func (service Service) loadCandidate() (Layout, State, error) {
	layout, err := service.validatedLayout()
	if err != nil {
		return Layout{}, State{}, err
	}
	info, err := os.Lstat(layout.Root)
	if err != nil {
		return Layout{}, State{}, fmt.Errorf("inspect candidate experience: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return Layout{}, State{}, fmt.Errorf("candidate experience root must be a real directory: %s", layout.Root)
	}
	state, err := readState(layout)
	if err != nil {
		return Layout{}, State{}, err
	}
	return layout, state, nil
}
