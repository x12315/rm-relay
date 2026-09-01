package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"github.com/x12315/rm-relay/internal/execution/command"
	"github.com/x12315/rm-relay/tests/support/candidate/internal/candidate"
)

type binaryBuilder struct {
	runner command.Runner
	root   string
}

func (builder binaryBuilder) BuildHostBinary(ctx context.Context, output string) (candidate.Binary, error) {
	revisionResult, err := builder.runner.Run(ctx, command.Request{Name: "git", Arguments: []string{"rev-parse", "--short=12", "HEAD"}, Directory: builder.root})
	if err != nil {
		return candidate.Binary{}, fmt.Errorf("read candidate revision: %w", err)
	}
	version := "candidate-" + strings.TrimSpace(revisionResult.Stdout)
	result, err := builder.runner.Run(ctx, command.Request{
		Name: "go", Arguments: []string{"build", "-trimpath", "-ldflags", "-s -w -X main.version=" + version, "-o", output, "./cmd/rm-relay"},
		Directory: builder.root, Stderr: os.Stderr,
	})
	if err != nil {
		return candidate.Binary{}, fmt.Errorf("build candidate CLI: %w: %s", err, strings.TrimSpace(result.Stderr))
	}
	versionResult, err := builder.runner.Run(ctx, command.Request{Name: output, Arguments: []string{"--version"}})
	if err != nil {
		return candidate.Binary{}, fmt.Errorf("read candidate CLI version: %w", err)
	}
	actualVersion := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(versionResult.Stdout), "rm-relay version "))
	if actualVersion != version {
		return candidate.Binary{}, fmt.Errorf("candidate CLI version = %q, want %q", actualVersion, version)
	}
	file, err := os.Open(output)
	if err != nil {
		return candidate.Binary{}, err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return candidate.Binary{}, err
	}
	return candidate.Binary{Path: output, Version: actualVersion, SHA256: hex.EncodeToString(hash.Sum(nil))}, nil
}

func main() {
	ctx := context.Background()
	runner := command.OSRunner{}
	repositoryRoot, err := locateRepositoryRoot(ctx, runner)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	cacheRoot, err := os.UserCacheDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	service := candidate.Service{RepositoryRoot: repositoryRoot, UserCacheRoot: cacheRoot, Runner: runner, BinaryBuilder: binaryBuilder{runner: runner, root: repositoryRoot}, Shell: shell(), Stdin: os.Stdin, Stdout: os.Stdout, Stderr: os.Stderr}
	root := &cobra.Command{Use: "rm-relay-candidate", SilenceUsage: true, SilenceErrors: true}
	root.AddCommand(&cobra.Command{Use: "prepare", Args: cobra.NoArgs, RunE: func(command *cobra.Command, _ []string) error {
		prepared, err := service.Prepare(command.Context())
		if err == nil {
			fmt.Printf("Candidate environment: %s\n", prepared.Root)
		}
		return err
	}})
	root.AddCommand(&cobra.Command{Use: "enter", Args: cobra.NoArgs, RunE: func(command *cobra.Command, _ []string) error { return service.Enter(command.Context()) }})
	root.AddCommand(&cobra.Command{Use: "clean", Args: cobra.NoArgs, RunE: func(command *cobra.Command, _ []string) error { return service.Clean(command.Context()) }})
	if err := root.ExecuteContext(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "rm-relay-candidate: %s\n", err)
		os.Exit(1)
	}
}

func locateRepositoryRoot(ctx context.Context, runner command.Runner) (string, error) {
	result, err := runner.Run(ctx, command.Request{Name: "git", Arguments: []string{"rev-parse", "--show-toplevel"}})
	if err != nil {
		return "", fmt.Errorf("locate repository root: %w", err)
	}
	root := strings.TrimSpace(result.Stdout)
	if root == "" {
		return "", fmt.Errorf("Git returned an empty repository root")
	}
	return root, nil
}

func shell() string {
	if runtime.GOOS == "windows" {
		if value := os.Getenv("COMSPEC"); value != "" {
			return value
		}
		return "cmd.exe"
	}
	if value := os.Getenv("SHELL"); value != "" {
		return value
	}
	return "/bin/sh"
}
