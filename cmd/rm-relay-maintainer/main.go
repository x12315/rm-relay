package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"strings"

	"github.com/x12315/rm-relay/internal/execution/command"
	"github.com/x12315/rm-relay/internal/maintainer"
	"github.com/x12315/rm-relay/internal/maintainer/distribution"
	"github.com/x12315/rm-relay/internal/maintainer/experience"
)

func main() {
	ctx, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stopSignals()
	runner := command.OSRunner{}
	repositoryRoot, err := locateRepositoryRoot(ctx, runner)
	if err != nil {
		fmt.Fprintf(os.Stderr, "rm-relay-maintainer: %s\n", err)
		os.Exit(1)
	}
	userCacheRoot, err := os.UserCacheDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "rm-relay-maintainer: locate user cache: %s\n", err)
		os.Exit(1)
	}
	packager := distribution.Packager{
		Runner:         runner,
		RepositoryRoot: repositoryRoot,
		GoReleaser:     "goreleaser",
		Progress:       os.Stderr,
	}
	experienceService := experience.Service{
		RepositoryRoot: repositoryRoot,
		UserCacheRoot:  userCacheRoot,
		Runner:         runner,
		BinaryBuilder:  packager,
		Shell:          resolveShell(runtime.GOOS, os.Getenv),
		Stdin:          os.Stdin,
		Stdout:         os.Stdout,
		Stderr:         os.Stderr,
	}
	exitCode := maintainer.Run(ctx, os.Args[1:], maintainer.Actions{
		CrossBuild:      packager.CrossBuild,
		PackageSnapshot: packager.PackageSnapshot,
		Prepare:         experienceService.Prepare,
		Enter:           experienceService.Enter,
		Clean:           experienceService.Clean,
	}, maintainer.Streams{Stdin: os.Stdin, Stdout: os.Stdout, Stderr: os.Stderr}, os.Getenv)
	os.Exit(exitCode)
}

func locateRepositoryRoot(ctx context.Context, runner command.Runner) (string, error) {
	result, err := runner.Run(ctx, command.Request{Name: "git", Arguments: []string{"rev-parse", "--show-toplevel"}})
	if err != nil {
		return "", fmt.Errorf("locate repository root: %w", err)
	}
	repositoryRoot := strings.TrimSpace(result.Stdout)
	if repositoryRoot == "" {
		return "", fmt.Errorf("locate repository root: Git returned an empty path")
	}
	return repositoryRoot, nil
}

func resolveShell(goos string, getenv func(string) string) string {
	if goos == "windows" {
		if shell := getenv("COMSPEC"); shell != "" {
			return shell
		}
		return "cmd.exe"
	}
	if shell := getenv("SHELL"); shell != "" {
		return shell
	}
	return "/bin/sh"
}
