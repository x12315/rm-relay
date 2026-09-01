package environment

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/x12315/rm-relay/internal/execution/buildx"
)

type recordingBuildExecutor struct {
	request buildx.BuildRequest
	content string
	err     error
}

func (executor *recordingBuildExecutor) Build(_ context.Context, request buildx.BuildRequest) error {
	executor.request = request
	if executor.err != nil {
		return executor.err
	}
	return os.WriteFile(filepath.Join(request.OutputDirectory, "identity.toml"), []byte(executor.content), 0o600)
}

func TestBuildKitVerifierExportsAndParsesIdentity(t *testing.T) {
	reference := "registry.example.org/rm-relay/embedded@sha256:" + strings.Repeat("a", 64)
	executor := &recordingBuildExecutor{content: "schema_version = 1\nid = \"embedded-development\"\n"}
	verifier := BuildKitVerifier{Buildx: executor, Progress: io.Discard}

	identity, err := verifier.Verify(context.Background(), "rm-relay-local-workspace-buildx", reference)
	if err != nil {
		t.Fatal(err)
	}
	if identity.ID != "embedded-development" {
		t.Fatalf("identity = %#v", identity)
	}
	if executor.request.Builder != "rm-relay-local-workspace-buildx" || executor.request.BuildArguments["RM_RELAY_ENVIRONMENT"] != reference {
		t.Fatalf("Build() request = %#v", executor.request)
	}
	dockerfile, err := io.ReadAll(executor.request.Dockerfile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(dockerfile), IdentityPath) {
		t.Fatalf("identity Dockerfile does not export %s", IdentityPath)
	}
}

func TestBuildKitVerifierRejectsMutableReferenceBeforeBuild(t *testing.T) {
	executor := &recordingBuildExecutor{}
	_, err := (BuildKitVerifier{Buildx: executor}).Verify(context.Background(), "rm-relay-local-workspace-buildx", "registry.example/image:latest")
	if err == nil {
		t.Fatal("Verify() accepted mutable reference")
	}
	if executor.request.Builder != "" {
		t.Fatal("Build() executed for invalid reference")
	}
}
