package environment

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/x12315/rm-relay/internal/execution/buildx"
)

//go:embed identity.Dockerfile
var identityDockerfile []byte

// Verifier proves that one Builder can pull an image and read its RM Relay identity.
type Verifier interface {
	Verify(context.Context, string, string) (Identity, error)
}

type buildExecutor interface {
	Build(context.Context, buildx.BuildRequest) error
}

// BuildKitVerifier exports the identity file through the selected Buildx Builder.
type BuildKitVerifier struct {
	Buildx   buildExecutor
	Progress io.Writer
}

// Verify pulls a digest-pinned image through builderName and returns its strict identity.
func (verifier BuildKitVerifier) Verify(ctx context.Context, builderName, reference string) (Identity, error) {
	if builderName == "" {
		return Identity{}, fmt.Errorf("Buildx builder name is required")
	}
	if _, err := ParseDigestReference(reference); err != nil {
		return Identity{}, err
	}
	if verifier.Buildx == nil {
		return Identity{}, fmt.Errorf("Buildx client is not configured")
	}
	workspace, err := os.MkdirTemp("", "rm-relay-environment-check-*")
	if err != nil {
		return Identity{}, fmt.Errorf("create environment verification workspace: %w", err)
	}
	defer os.RemoveAll(workspace)
	contextDirectory := filepath.Join(workspace, "context")
	outputDirectory := filepath.Join(workspace, "output")
	if err := os.Mkdir(contextDirectory, 0o700); err != nil {
		return Identity{}, fmt.Errorf("create environment verification context: %w", err)
	}
	if err := os.Mkdir(outputDirectory, 0o700); err != nil {
		return Identity{}, fmt.Errorf("create environment verification output: %w", err)
	}
	request := buildx.BuildRequest{
		Builder:          builderName,
		ContextDirectory: contextDirectory,
		OutputDirectory:  outputDirectory,
		Dockerfile:       bytes.NewReader(identityDockerfile),
		BuildArguments:   map[string]string{"RM_RELAY_ENVIRONMENT": reference},
		Progress:         verifier.Progress,
	}
	if err := verifier.Buildx.Build(ctx, request); err != nil {
		return Identity{}, fmt.Errorf("verify environment through Builder %q: %w", builderName, err)
	}
	identityPath := filepath.Join(outputDirectory, "identity.toml")
	info, err := os.Lstat(identityPath)
	if err != nil {
		return Identity{}, fmt.Errorf("read exported environment identity: %w", err)
	}
	if !info.Mode().IsRegular() {
		return Identity{}, fmt.Errorf("exported environment identity must be a regular file")
	}
	content, err := os.ReadFile(identityPath)
	if err != nil {
		return Identity{}, fmt.Errorf("read exported environment identity: %w", err)
	}
	return ParseIdentity(content)
}
