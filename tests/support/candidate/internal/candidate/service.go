package candidate

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/x12315/rm-relay/internal/builder"
	"github.com/x12315/rm-relay/internal/execution/buildx"
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
	Root        string
	Revision    string
	CLIVersion  string
	ImageID     string
	TemplateURL string
}

// Service manages one repository's disposable candidate experience environment.
type Service struct {
	RepositoryRoot string
	UserCacheRoot  string
	Runner         command.Runner
	BinaryBuilder  BinaryBuilder
	Now            func() time.Time
	Shell          string
	Stdin          io.Reader
	Stdout         io.Writer
	Stderr         io.Writer
}

// Prepare creates a candidate CLI, image, template origin and empty workspace outside the repository.
func (service Service) Prepare(ctx context.Context) (prepared Prepared, returnError error) {
	layout, err := service.validatedLayout()
	if err != nil {
		return Prepared{}, err
	}
	if service.BinaryBuilder == nil {
		return Prepared{}, fmt.Errorf("candidate CLI builder is not configured")
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
	previousImageID, err := service.currentTaggedImage(ctx)
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
	imageChanged := false
	published := false
	defer func() {
		if published {
			return
		}
		_ = os.RemoveAll(preparingRoot)
		if imageChanged {
			returnError = errors.Join(returnError, service.restoreImage(ctx, previousImageID))
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
	imageChanged = true
	imageID, err := service.buildDevelopmentImage(ctx, layout.RepositoryRoot)
	if err != nil {
		return Prepared{}, err
	}

	now := time.Now
	if service.Now != nil {
		now = service.Now
	}
	state := State{
		SchemaVersion:    StateSchemaVersion,
		Marker:           managedStateMarker,
		RepositoryRoot:   layout.RepositoryRoot,
		RepositoryKey:    layout.RepositoryKey,
		Revision:         revision,
		CLIVersion:       binary.Version,
		CLISHA256:        binary.SHA256,
		ImageReference:   developmentImageReference,
		ImageID:          imageID,
		PreviousImageID:  previousImageID,
		TemplateRevision: templateRevision,
		CreatedAt:        now().UTC(),
	}
	if err := writeState(preparingLayout, state); err != nil {
		return Prepared{}, err
	}
	if err := os.Rename(preparingRoot, layout.Root); err != nil {
		return Prepared{}, fmt.Errorf("publish candidate experience: %w", err)
	}
	published = true
	return Prepared{
		Root:        layout.Root,
		Revision:    revision,
		CLIVersion:  binary.Version,
		ImageID:     imageID,
		TemplateURL: templateURL(layout.TemplateOrigin),
	}, nil
}

// Enter validates candidate identities and opens an interactive shell in its empty workspace.
func (service Service) Enter(ctx context.Context) error {
	layout, state, err := service.loadCandidate()
	if err != nil {
		return err
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
	imageID, err := service.inspectImage(ctx)
	if err != nil {
		return err
	}
	if imageID != state.ImageID {
		return fmt.Errorf("candidate image identity changed: got %s, want %s", imageID, state.ImageID)
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
		fmt.Fprintf(service.Stdout, "Candidate revision: %s\nCandidate CLI: %s\nDevelopment image: %s\nTemplate: %s\nClone with: git clone \"$RM_RELAY_TEMPLATE_URL\" project\n", state.Revision, state.CLIVersion, state.ImageID, templateURL(layout.TemplateOrigin))
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

// Clean restores the previous development-image tag before deleting the managed candidate directory.
func (service Service) Clean(ctx context.Context) error {
	layout, state, err := service.loadCandidate()
	if err != nil {
		return err
	}
	configuredBuilders, err := (builder.Store{Directory: filepath.Join(layout.ConfigDirectory, "rm-relay")}).Load()
	if err != nil {
		return fmt.Errorf("load candidate Builder catalog: %w", err)
	}
	builderManager := builder.Service{
		Store:  builder.Store{Directory: filepath.Join(layout.ConfigDirectory, "rm-relay")},
		Buildx: buildx.CLI{Runner: service.Runner},
	}
	for _, definition := range configuredBuilders {
		if err := builderManager.Remove(ctx, definition.ID); err != nil {
			return fmt.Errorf("remove candidate Builder %q: %w", definition.ID, err)
		}
	}
	if err := service.restoreImage(ctx, state.PreviousImageID); err != nil {
		return err
	}
	if err := os.RemoveAll(layout.Root); err != nil {
		return fmt.Errorf("remove candidate experience: %w", err)
	}
	return nil
}

func (service Service) validatedLayout() (Layout, error) {
	if service.Runner == nil {
		return Layout{}, fmt.Errorf("process runner is not configured")
	}
	return ResolveLayout(service.RepositoryRoot, service.UserCacheRoot)
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
