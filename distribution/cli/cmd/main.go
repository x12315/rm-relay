package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/x12315/rm-relay/distribution/cli/internal/packager"
	"github.com/x12315/rm-relay/internal/execution/command"
)

func main() {
	runner := command.OSRunner{}
	rootPath, err := repositoryRoot(context.Background(), runner)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	service := packager.Packager{Runner: runner, RepositoryRoot: rootPath, GoReleaser: "goreleaser", Progress: os.Stderr}
	root := &cobra.Command{Use: "rm-relay-distribution", SilenceUsage: true, SilenceErrors: true}
	root.AddCommand(outputCommand("cross-build", service.CrossBuild), outputCommand("snapshot", service.PackageSnapshot))
	root.AddCommand(&cobra.Command{Use: "host-binary <absolute-output-path>", Args: cobra.ExactArgs(1), RunE: func(command *cobra.Command, arguments []string) error {
		_, err := service.BuildHostBinary(command.Context(), arguments[0])
		return err
	}})
	if err := root.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "rm-relay-distribution: %s\n", err)
		os.Exit(1)
	}
}

func outputCommand(name string, action func(context.Context, string) error) *cobra.Command {
	return &cobra.Command{Use: name, Args: cobra.NoArgs, RunE: func(command *cobra.Command, _ []string) error {
		output := os.Getenv("RM_RELAY_CLI_OUTPUT_DIR")
		if output == "" {
			return fmt.Errorf("RM_RELAY_CLI_OUTPUT_DIR must name an absolute directory outside the repository")
		}
		return action(command.Context(), output)
	}}
}

func repositoryRoot(ctx context.Context, runner command.Runner) (string, error) {
	result, err := runner.Run(ctx, command.Request{Name: "git", Arguments: []string{"rev-parse", "--show-toplevel"}})
	if err != nil {
		return "", fmt.Errorf("locate repository root: %w", err)
	}
	root := strings.TrimSpace(result.Stdout)
	if root == "" {
		return "", fmt.Errorf("locate repository root: Git returned an empty path")
	}
	return root, nil
}
