package candidate

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/x12315/rm-relay/internal/builder"
	"github.com/x12315/rm-relay/internal/execution/command"
)

func TestPrepareRejectsDirtyRepository(t *testing.T) {
	fixture := newServiceFixture(t)
	fixture.runner.dirty = true

	_, err := fixture.service.Prepare(context.Background())

	if err == nil || !strings.Contains(err.Error(), "uncommitted") {
		t.Fatalf("Prepare() error = %v", err)
	}
	if _, statError := os.Stat(fixture.layout.Root); !os.IsNotExist(statError) {
		t.Fatalf("candidate root exists after rejected prepare: %v", statError)
	}
}

func TestPrepareCreatesManagedCandidateWithoutChangingSourceRefs(t *testing.T) {
	fixture := newServiceFixture(t)

	prepared, err := fixture.service.Prepare(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	state, err := readState(fixture.layout)
	if err != nil {
		t.Fatal(err)
	}
	if prepared.Root != fixture.layout.Root || state.Revision != fixture.runner.repositoryRevision {
		t.Fatalf("prepared candidate = %#v state = %#v", prepared, state)
	}
	if state.BuilderID != fixture.service.Builder.ID || state.BuilderKind != fixture.service.Builder.Kind || state.BuildxBuilder != fixture.service.Builder.BuildxBuilder {
		t.Fatalf("Builder identity = %#v", state)
	}
	if state.EnvironmentID != fixture.service.EnvironmentID || state.EnvironmentReference != fixture.service.EnvironmentReference {
		t.Fatalf("environment identity = %q %q", state.EnvironmentID, state.EnvironmentReference)
	}
	if _, err := os.Stat(fixture.layout.TemplateOrigin); err != nil {
		t.Fatalf("template origin was not created: %v", err)
	}
	for _, request := range fixture.runner.requests {
		if request.Name == "docker" || request.Name == "mise" {
			t.Fatalf("prepare mutated environment infrastructure: %#v", request)
		}
		joined := strings.Join(request.Arguments, " ")
		if strings.Contains(joined, "subtree") || strings.Contains(joined, " branch ") {
			t.Fatalf("prepare changed source refs: %#v", request)
		}
	}
	configuredBuilders, err := (builder.Store{Directory: filepath.Join(fixture.layout.ConfigDirectory, "rm-relay")}).Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(configuredBuilders) != 1 || configuredBuilders[0].ID != fixture.service.Builder.ID {
		t.Fatalf("isolated Builder catalog = %#v", configuredBuilders)
	}
	wantReference, err := configuredBuilders[0].EnvironmentReference(fixture.service.EnvironmentID)
	if err != nil || wantReference != fixture.service.EnvironmentReference {
		t.Fatalf("isolated environment reference = %q, %v", wantReference, err)
	}
}

func TestPrepareCopiesSelectedRemoteBuilderWithoutOwningIt(t *testing.T) {
	fixture := newServiceFixture(t)
	fixture.service.Builder = builder.Definition{
		ID:            "team",
		Kind:          builder.KindRemoteBuildKit,
		BuildxBuilder: "rm-relay-team",
		Environments:  map[string]string{"other": "registry.example/other@sha256:" + strings.Repeat("c", 64)},
	}

	if _, err := fixture.service.Prepare(context.Background()); err != nil {
		t.Fatal(err)
	}
	definitions, err := (builder.Store{Directory: filepath.Join(fixture.layout.ConfigDirectory, "rm-relay")}).Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(definitions) != 1 || definitions[0].ID != "team" || definitions[0].BuildxBuilder != "rm-relay-team" {
		t.Fatalf("isolated Builder catalog = %#v", definitions)
	}
	if definitions[0].Environments["other"] == "" || definitions[0].Environments[fixture.service.EnvironmentID] != fixture.service.EnvironmentReference {
		t.Fatalf("isolated Builder environments = %#v", definitions[0].Environments)
	}
}

func TestEnterValidatesCandidateThenOpensShellWithoutCloning(t *testing.T) {
	fixture := newServiceFixture(t)
	if _, err := fixture.service.Prepare(context.Background()); err != nil {
		t.Fatal(err)
	}
	fixture.runner.requests = nil

	if err := fixture.service.Enter(context.Background()); err != nil {
		t.Fatal(err)
	}

	var shellRequest *command.Request
	for index := range fixture.runner.requests {
		request := &fixture.runner.requests[index]
		if request.Name == fixture.service.Shell {
			shellRequest = request
		}
		if request.Name == "git" && len(request.Arguments) > 0 && request.Arguments[0] == "clone" {
			t.Fatalf("enter cloned a project: %#v", request)
		}
	}
	if shellRequest == nil || !shellRequest.Interactive || shellRequest.Directory != fixture.layout.Workspace {
		t.Fatalf("interactive shell request = %#v", shellRequest)
	}
	if !strings.HasPrefix(shellRequest.Environment["PATH"], fixture.layout.BinaryDirectory+string(os.PathListSeparator)) {
		t.Fatalf("candidate PATH = %q", shellRequest.Environment["PATH"])
	}
	if !strings.HasPrefix(shellRequest.Environment["RM_RELAY_TEMPLATE_URL"], "file://") {
		t.Fatalf("template URL = %q", shellRequest.Environment["RM_RELAY_TEMPLATE_URL"])
	}
	if shellRequest.Environment["RM_RELAY_CONFIG_DIR"] != fixture.layout.ConfigDirectory {
		t.Fatalf("candidate config directory = %q", shellRequest.Environment["RM_RELAY_CONFIG_DIR"])
	}
	if !strings.Contains(fixture.service.Stdout.(*bytes.Buffer).String(), "git clone") {
		t.Fatalf("enter instructions = %q", fixture.service.Stdout.(*bytes.Buffer).String())
	}
}

func TestEnterRejectsChangedBinary(t *testing.T) {
	fixture := newServiceFixture(t)
	if _, err := fixture.service.Prepare(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fixture.layout.BinaryPath, []byte("changed"), 0o755); err != nil {
		t.Fatal(err)
	}

	err := fixture.service.Enter(context.Background())

	if err == nil || !strings.Contains(err.Error(), "CLI identity") {
		t.Fatalf("Enter() error = %v", err)
	}
}

func TestEnterRejectsChangedEnvironmentMapping(t *testing.T) {
	fixture := newServiceFixture(t)
	if _, err := fixture.service.Prepare(context.Background()); err != nil {
		t.Fatal(err)
	}
	changed := fixture.service.Builder
	changed.Environments = map[string]string{fixture.service.EnvironmentID: "registry.example/changed@sha256:" + strings.Repeat("c", 64)}
	store := builder.Store{Directory: filepath.Join(fixture.layout.ConfigDirectory, "rm-relay")}
	if err := store.Save([]builder.Definition{changed}); err != nil {
		t.Fatal(err)
	}

	err := fixture.service.Enter(context.Background())

	if err == nil || !strings.Contains(err.Error(), "environment identity changed") {
		t.Fatalf("Enter() error = %v", err)
	}
}

func TestCleanRemovesOnlyCandidateOwnedFiles(t *testing.T) {
	fixture := newServiceFixture(t)
	if _, err := fixture.service.Prepare(context.Background()); err != nil {
		t.Fatal(err)
	}
	fixture.runner.requests = nil
	fixture.service.Runner = nil

	if err := fixture.service.Clean(context.Background()); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(fixture.layout.Root); !os.IsNotExist(err) {
		t.Fatalf("candidate root still exists: %v", err)
	}
	if len(fixture.runner.requests) != 0 {
		t.Fatalf("clean touched borrowed resources: %#v", fixture.runner.requests)
	}
}

func TestCleanRejectsInvalidStateBeforeRemovingFiles(t *testing.T) {
	fixture := newServiceFixture(t)
	if _, err := fixture.service.Prepare(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fixture.layout.StatePath, []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := fixture.service.Clean(context.Background())

	if err == nil || !strings.Contains(err.Error(), "schema version") {
		t.Fatalf("Clean() error = %v", err)
	}
	if _, statError := os.Stat(fixture.layout.Root); statError != nil {
		t.Fatalf("candidate root was removed after rejected clean: %v", statError)
	}
}

type serviceFixture struct {
	service Service
	layout  Layout
	runner  *serviceRunner
}

func newServiceFixture(t *testing.T) serviceFixture {
	t.Helper()
	repositoryRoot := filepath.Join(t.TempDir(), "repository")
	templatePath := filepath.Join(repositoryRoot, "project-templates", "cross-platform-cpp", "rm-relay.toml")
	if err := os.MkdirAll(filepath.Dir(templatePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(templatePath, []byte("schema_version = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cacheRoot := t.TempDir()
	layout, err := ResolveLayout(repositoryRoot, cacheRoot)
	if err != nil {
		t.Fatal(err)
	}
	runner := &serviceRunner{
		repositoryRevision: "0123456789abcdef0123456789abcdef01234567",
		templateRevision:   "abcdef0123456789abcdef0123456789abcdef01",
		templateFile:       "project-templates/cross-platform-cpp/rm-relay.toml",
	}
	candidateCLIBuilder := &binaryBuilder{version: "0.0.0-SNAPSHOT-test"}
	service := Service{
		RepositoryRoot:       repositoryRoot,
		UserCacheRoot:        cacheRoot,
		Builder:              builder.Definition{ID: builder.LocalID, Kind: builder.KindLocalBuildKit, BuildxBuilder: builder.LocalBuildxBuilder, Environments: map[string]string{}},
		EnvironmentID:        "embedded-development",
		EnvironmentReference: "registry.example/environment@sha256:" + strings.Repeat("a", 64),
		Runner:               runner,
		BinaryBuilder:        candidateCLIBuilder,
		Now:                  func() time.Time { return time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC) },
		Shell:                "candidate-shell",
		Stdin:                strings.NewReader(""),
		Stdout:               &bytes.Buffer{},
		Stderr:               &bytes.Buffer{},
	}
	return serviceFixture{service: service, layout: layout, runner: runner}
}

type binaryBuilder struct {
	version string
}

func (builder *binaryBuilder) BuildHostBinary(_ context.Context, outputPath string) (Binary, error) {
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return Binary{}, err
	}
	if err := os.WriteFile(outputPath, []byte("candidate CLI"), 0o755); err != nil {
		return Binary{}, err
	}
	digest, err := fileSHA256(outputPath)
	if err != nil {
		return Binary{}, err
	}
	return Binary{Path: outputPath, Version: builder.version, SHA256: digest}, nil
}

type serviceRunner struct {
	requests           []command.Request
	dirty              bool
	repositoryRevision string
	templateRevision   string
	templateFile       string
}

func (runner *serviceRunner) Run(_ context.Context, request command.Request) (command.Result, error) {
	runner.requests = append(runner.requests, request)
	if request.Name == "candidate-shell" {
		return command.Result{}, nil
	}
	if strings.HasSuffix(request.Name, "rm-relay") || strings.HasSuffix(request.Name, "rm-relay.exe") {
		return command.Result{Stdout: "rm-relay version 0.0.0-SNAPSHOT-test\n"}, nil
	}
	if request.Name != "git" {
		return command.Result{}, fmt.Errorf("unexpected command %q", request.Name)
	}
	return runner.runGit(request)
}

func (runner *serviceRunner) runGit(request command.Request) (command.Result, error) {
	joined := strings.Join(request.Arguments, " ")
	switch {
	case joined == "status --porcelain":
		if runner.dirty {
			return command.Result{Stdout: " M changed.go\n"}, nil
		}
		return command.Result{}, nil
	case joined == "rev-parse HEAD":
		if strings.Contains(request.Directory, ".preparing-") {
			return command.Result{Stdout: runner.templateRevision + "\n"}, nil
		}
		return command.Result{Stdout: runner.repositoryRevision + "\n"}, nil
	case joined == "ls-files -z -- project-templates/cross-platform-cpp":
		return command.Result{Stdout: runner.templateFile + "\x00"}, nil
	case strings.HasPrefix(joined, "clone --quiet --bare"):
		destination := request.Arguments[len(request.Arguments)-1]
		return command.Result{}, os.MkdirAll(destination, 0o755)
	case strings.HasPrefix(joined, "--git-dir") && strings.HasSuffix(joined, "rev-parse HEAD"):
		return command.Result{Stdout: runner.templateRevision + "\n"}, nil
	case joined == "init --quiet", strings.HasPrefix(joined, "config "), joined == "add .", strings.HasPrefix(joined, "commit --quiet"):
		return command.Result{}, nil
	default:
		return command.Result{}, fmt.Errorf("unexpected Git arguments %q", joined)
	}
}
