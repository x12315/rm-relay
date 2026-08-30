// Package maintainer exposes repository-only operations through a stable command tree.
package maintainer

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"github.com/x12315/rm-relay/internal/maintainer/experience"
)

// Actions connects maintainer commands to distribution and candidate-environment services.
type Actions struct {
	CrossBuild      func(context.Context, string) error
	PackageSnapshot func(context.Context, string) error
	Prepare         func(context.Context) (experience.Prepared, error)
	Enter           func(context.Context) error
	Clean           func(context.Context) error
}

// Streams are the maintainer command's user-facing process streams.
type Streams struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

// Run executes one maintainer command and returns its process exit code.
func Run(ctx context.Context, arguments []string, actions Actions, streams Streams, getenv func(string) string) int {
	root := &cobra.Command{
		Use:           "rm-relay-maintainer",
		Short:         "Maintain RM Relay distribution and candidate test assets",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.SetArgs(arguments)
	root.SetIn(streams.Stdin)
	root.SetOut(streams.Stdout)
	root.SetErr(streams.Stderr)
	root.AddCommand(newCLICommand(actions, streams.Stdout, getenv), newExperienceCommand(actions, streams.Stdout))
	if err := root.ExecuteContext(ctx); err != nil {
		fmt.Fprintf(streams.Stderr, "rm-relay-maintainer: %s\n", err)
		return 1
	}
	return 0
}

func newCLICommand(actions Actions, stdout io.Writer, getenv func(string) string) *cobra.Command {
	command := &cobra.Command{Use: "cli", Short: "Build local CLI candidates"}
	command.AddCommand(
		newCLIOutputCommand("cross-build", actions.CrossBuild, stdout, getenv),
		newCLIOutputCommand("package-snapshot", actions.PackageSnapshot, stdout, getenv),
	)
	return command
}

func newCLIOutputCommand(name string, action func(context.Context, string) error, stdout io.Writer, getenv func(string) string) *cobra.Command {
	return &cobra.Command{
		Use:  name,
		Args: cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			outputDirectory := getenv("RM_RELAY_CLI_OUTPUT_DIR")
			if outputDirectory == "" {
				return fmt.Errorf("RM_RELAY_CLI_OUTPUT_DIR must name an absolute directory outside the repository")
			}
			if action == nil {
				return fmt.Errorf("%s action is not configured", name)
			}
			if err := action(command.Context(), outputDirectory); err != nil {
				return err
			}
			fmt.Fprintf(stdout, "CLI output: %s\n", outputDirectory)
			return nil
		},
	}
}

func newExperienceCommand(actions Actions, stdout io.Writer) *cobra.Command {
	command := &cobra.Command{Use: "experience", Short: "Manage one local candidate experience environment"}
	command.AddCommand(
		&cobra.Command{
			Use:  "prepare",
			Args: cobra.NoArgs,
			RunE: func(command *cobra.Command, _ []string) error {
				if actions.Prepare == nil {
					return fmt.Errorf("prepare action is not configured")
				}
				prepared, err := actions.Prepare(command.Context())
				if err != nil {
					return err
				}
				fmt.Fprintf(stdout, "Candidate environment: %s\nRevision: %s\nCLI: %s\nDevelopment image: %s\nTemplate: %s\nNext: mise run experience:enter\n", prepared.Root, prepared.Revision, prepared.CLIVersion, prepared.ImageID, prepared.TemplateURL)
				return nil
			},
		},
		&cobra.Command{
			Use:  "enter",
			Args: cobra.NoArgs,
			RunE: func(command *cobra.Command, _ []string) error {
				if actions.Enter == nil {
					return fmt.Errorf("enter action is not configured")
				}
				return actions.Enter(command.Context())
			},
		},
		&cobra.Command{
			Use:  "clean",
			Args: cobra.NoArgs,
			RunE: func(command *cobra.Command, _ []string) error {
				if actions.Clean == nil {
					return fmt.Errorf("clean action is not configured")
				}
				if err := actions.Clean(command.Context()); err != nil {
					return err
				}
				fmt.Fprintln(stdout, "Candidate environment removed; shared build caches were not changed.")
				return nil
			},
		},
	)
	return command
}
