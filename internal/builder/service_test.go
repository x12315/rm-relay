package builder

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/x12315/rm-relay/internal/execution/buildx"
	"github.com/x12315/rm-relay/internal/execution/docker"
)

type fakeBuildx struct {
	created, removed []string
	createRequest    buildx.CreateRemoteRequest
	localRequest     buildx.CreateLocalRequest
	listed           []buildx.BuilderSummary
	removeError      error
	inspectCallCount int
}

func (client *fakeBuildx) ListBuilders(context.Context) ([]buildx.BuilderSummary, error) {
	return append([]buildx.BuilderSummary(nil), client.listed...), nil
}
func (client *fakeBuildx) CreateLocal(_ context.Context, request buildx.CreateLocalRequest) error {
	client.created = append(client.created, request.Name)
	client.localRequest = request
	client.listed = append(client.listed, buildx.BuilderSummary{Name: request.Name, Driver: "docker-container"})
	return nil
}

func (client *fakeBuildx) CreateRemote(_ context.Context, request buildx.CreateRemoteRequest) error {
	client.created = append(client.created, request.Name)
	client.createRequest = request
	return nil
}
func (client *fakeBuildx) RemoveBuilder(_ context.Context, name string) error {
	client.removed = append(client.removed, name)
	return client.removeError
}
func (client *fakeBuildx) InspectBuilder(context.Context, string) error {
	client.inspectCallCount++
	return nil
}
func (*fakeBuildx) Build(_ context.Context, request buildx.BuildRequest) error {
	return os.WriteFile(filepath.Join(request.OutputDirectory, "probe"), []byte("rm-relay\n"), 0o600)
}

type fakeDocker struct{}

func (fakeDocker) CheckEngine(context.Context) error                    { return nil }
func (fakeDocker) InspectImage(context.Context, string) (string, error) { return "", nil }
func (fakeDocker) Run(context.Context, docker.RunRequest) error         { return nil }
func (fakeDocker) TagImage(context.Context, string, string) error       { return nil }
func (fakeDocker) RemoveImage(context.Context, string) error            { return nil }
func TestAddCreatesBuildxThenPersistsLogicalDefinition(t *testing.T) {
	root := t.TempDir()
	paths := makeTLSFiles(t, root)
	buildxClient := &fakeBuildx{}
	service := Service{Store: Store{Directory: filepath.Join(root, "config")}, Buildx: buildxClient, Docker: fakeDocker{}}
	err := service.Add(context.Background(), AddRequest{ID: "team", Endpoint: "tcp://build.example.org:1234", CAPath: paths[0], CertificatePath: paths[1], KeyPath: paths[2], ServerName: "build.example.org"})
	if err != nil {
		t.Fatal(err)
	}
	definitions, err := service.Store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(definitions) != 1 || definitions[0].BuildxBuilder != "rm-relay-team" {
		t.Fatalf("definitions = %#v", definitions)
	}
	content, _ := os.ReadFile(filepath.Join(service.Store.Directory, FileName))
	for _, path := range paths {
		if strings.Contains(string(content), path) {
			t.Fatalf("catalog contains TLS path %q", path)
		}
	}
}

func TestSetEnvironmentRequiresImmutableDigest(t *testing.T) {
	store := Store{Directory: filepath.Join(t.TempDir(), "config")}
	if err := store.Save([]Definition{{ID: "team", Kind: KindRemoteBuildKit, BuildxBuilder: "rm-relay-team", Environments: map[string]string{}}}); err != nil {
		t.Fatal(err)
	}
	service := Service{Store: store}
	if err := service.SetEnvironment("team", "embedded-development", "registry/image:latest"); err == nil {
		t.Fatal("mutable image accepted")
	}
	reference := "registry/image@sha256:" + strings.Repeat("a", 64)
	if err := service.SetEnvironment("team", "embedded-development", reference); err != nil {
		t.Fatal(err)
	}
}

func TestSetEnvironmentCreatesPersistentLocalMapping(t *testing.T) {
	store := Store{Directory: filepath.Join(t.TempDir(), "config")}
	service := Service{Store: store}
	reference := "registry/image@sha256:" + strings.Repeat("b", 64)
	if err := service.SetEnvironment(LocalID, "embedded-development", reference); err != nil {
		t.Fatal(err)
	}
	definitions, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(definitions) != 1 || definitions[0].ID != LocalID || definitions[0].Environments["embedded-development"] != reference {
		t.Fatalf("definitions = %#v", definitions)
	}
}

func TestPrepareLocalCreatesPinnedBuilderOnce(t *testing.T) {
	client := &fakeBuildx{}
	service := Service{Store: Store{Directory: filepath.Join(t.TempDir(), "config")}, Buildx: client, Docker: fakeDocker{}}
	if err := service.Prepare(context.Background(), LocalID); err != nil {
		t.Fatal(err)
	}
	if err := service.Prepare(context.Background(), LocalID); err != nil {
		t.Fatal(err)
	}
	if len(client.created) != 1 || client.localRequest.Image != LocalBuildKitImage {
		t.Fatalf("created = %#v, request = %#v", client.created, client.localRequest)
	}
	if client.inspectCallCount != 1 {
		t.Fatalf("inspect calls = %d, want 1", client.inspectCallCount)
	}
}

func TestPrepareLocalRejectsForeignBuildxDriver(t *testing.T) {
	client := &fakeBuildx{listed: []buildx.BuilderSummary{{Name: LocalBuildxBuilder, Driver: "docker"}}}
	service := Service{Store: Store{Directory: filepath.Join(t.TempDir(), "config")}, Buildx: client, Docker: fakeDocker{}}
	if err := service.Prepare(context.Background(), LocalID); err == nil || !strings.Contains(err.Error(), "docker-container") {
		t.Fatalf("Prepare() error = %v", err)
	}
}

func TestRemoveDeletesMappingAndBuildxResource(t *testing.T) {
	store := Store{Directory: filepath.Join(t.TempDir(), "config")}
	definition := Definition{ID: "team", Kind: KindRemoteBuildKit, BuildxBuilder: "rm-relay-team", Environments: map[string]string{}}
	if err := store.Save([]Definition{definition}); err != nil {
		t.Fatal(err)
	}
	client := &fakeBuildx{}
	service := Service{Store: store, Buildx: client}

	if err := service.Remove(context.Background(), "team"); err != nil {
		t.Fatal(err)
	}
	definitions, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(definitions) != 0 || len(client.removed) != 1 || client.removed[0] != "rm-relay-team" {
		t.Fatalf("definitions = %#v, removed = %#v", definitions, client.removed)
	}
}

func TestRemoveRestoresMappingWhenBuildxRemovalFails(t *testing.T) {
	store := Store{Directory: filepath.Join(t.TempDir(), "config")}
	definition := Definition{ID: "team", Kind: KindRemoteBuildKit, BuildxBuilder: "rm-relay-team", Environments: map[string]string{}}
	if err := store.Save([]Definition{definition}); err != nil {
		t.Fatal(err)
	}
	client := &fakeBuildx{removeError: errors.New("builder is busy")}
	service := Service{Store: store, Buildx: client}

	err := service.Remove(context.Background(), "team")

	if err == nil || !strings.Contains(err.Error(), "builder is busy") {
		t.Fatalf("Remove() error = %v", err)
	}
	definitions, loadError := store.Load()
	if loadError != nil {
		t.Fatal(loadError)
	}
	if len(definitions) != 1 || definitions[0].ID != "team" {
		t.Fatalf("restored definitions = %#v", definitions)
	}
}

func makeTLSFiles(t *testing.T, root string) []string {
	t.Helper()
	paths := []string{filepath.Join(root, "ca.pem"), filepath.Join(root, "cert.pem"), filepath.Join(root, "key.pem")}
	for _, path := range paths {
		if err := os.WriteFile(path, []byte("fixture"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return paths
}
