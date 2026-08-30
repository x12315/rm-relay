// Package cli composes RM Relay modules into the public command line.
package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/x12315/rm-relay/internal/build"
	"github.com/x12315/rm-relay/internal/build/output"
	"github.com/x12315/rm-relay/internal/profile"
	"github.com/x12315/rm-relay/internal/project"
	"github.com/x12315/rm-relay/internal/target"
)

// Dependencies contains the module boundaries supplied by the executable composition root.
type Dependencies struct {
	Profiles        profile.Catalog
	BuildBackends   build.BackendCatalog
	DefaultBackend  string
	FlashAdapters   target.FlashAdapterCatalog
	ProducerVersion string
	Stdout          io.Writer
	Stderr          io.Writer
}

type application struct {
	dependencies Dependencies
	projectRoot  string
	outputFormat string
}

type commandResult struct {
	OK        bool     `json:"ok"`
	Operation string   `json:"operation"`
	ProjectID string   `json:"project_id,omitempty"`
	Profile   string   `json:"profile,omitempty"`
	Output    string   `json:"output,omitempty"`
	Command   []string `json:"command,omitempty"`
	Executed  *bool    `json:"executed,omitempty"`
}

type commandFailure struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type failureResult struct {
	OK        bool           `json:"ok"`
	Operation string         `json:"operation,omitempty"`
	Error     commandFailure `json:"error"`
}

type cliError struct {
	code      string
	operation string
	err       error
	exitCode  int
}

func (failure *cliError) Error() string {
	return failure.err.Error()
}

// Run executes one CLI command and writes either human text or one JSON result object.
func Run(ctx context.Context, arguments []string, dependencies Dependencies) int {
	if dependencies.Stdout == nil {
		dependencies.Stdout = io.Discard
	}
	if dependencies.Stderr == nil {
		dependencies.Stderr = io.Discard
	}
	app := &application{dependencies: dependencies, projectRoot: ".", outputFormat: "human"}
	root := app.newRootCommand()
	root.SetArgs(arguments)
	err := root.ExecuteContext(ctx)
	if err == nil {
		return 0
	}
	var failure *cliError
	if !errors.As(err, &failure) {
		failure = invalidArguments(root.CommandPath(), err)
	}
	emitFailure(app.outputFormat, dependencies, failure)
	return failure.exitCode
}

func (app *application) newRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:           "rm-relay",
		Short:         "RM 机器人开发基础设施统一入口",
		Version:       app.dependencies.ProducerVersion,
		SilenceErrors: true,
		SilenceUsage:  true,
		Args:          cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return invalidArguments("", fmt.Errorf("a command is required"))
		},
	}
	root.SetOut(app.dependencies.Stdout)
	root.SetErr(app.dependencies.Stderr)
	root.PersistentFlags().StringVar(&app.projectRoot, "project", ".", "项目根目录")
	root.PersistentFlags().StringVar(&app.outputFormat, "format", "human", "输出格式：human 或 json")
	root.PersistentPreRunE = func(command *cobra.Command, _ []string) error {
		if app.outputFormat != "human" && app.outputFormat != "json" {
			return invalidArguments(command.Name(), fmt.Errorf("unsupported output format %q", app.outputFormat))
		}
		return nil
	}
	root.SetFlagErrorFunc(func(command *cobra.Command, err error) error {
		return invalidArguments(command.Name(), err)
	})
	root.AddCommand(app.newInitCommand(), app.newBuildCommand(), app.newFlashCommand())
	return root
}

func (app *application) newInitCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "为当前项目建立稳定身份",
		Args:  noArguments("init"),
		RunE: func(_ *cobra.Command, _ []string) error {
			projectID, err := project.Initialize(app.projectRoot)
			if err != nil {
				return newCLIError("project_invalid", "init", 1, err)
			}
			emitSuccess(app.outputFormat, app.dependencies, commandResult{OK: true, Operation: "init", ProjectID: projectID})
			return nil
		},
	}
}

func (app *application) newBuildCommand() *cobra.Command {
	var profileOverride string
	command := &cobra.Command{
		Use:   "build",
		Short: "在受控开发环境中构建项目",
		Args:  noArguments("build"),
		RunE: func(command *cobra.Command, _ []string) error {
			plan, err := build.Resolve(build.OperationBuild, app.projectRoot, profileOverride, app.dependencies.Profiles)
			if err != nil {
				return resolutionFailure("build", err)
			}
			backend, err := app.dependencies.BuildBackends.Resolve(app.dependencies.DefaultBackend)
			if err != nil {
				return newCLIError("environment_invalid", "build", 1, err)
			}
			service := build.Service{
				Backend:         backend,
				ProducerVersion: app.dependencies.ProducerVersion,
			}
			if _, err := service.Execute(command.Context(), plan); err != nil {
				return newCLIError("build_failed", "build", 1, err)
			}
			manifestPath := filepath.Join(plan.OutputDirectory, output.ManifestFileName)
			relativeManifestPath, err := filepath.Rel(plan.ProjectRoot, manifestPath)
			if err != nil {
				return newCLIError("build_output_invalid", "build", 1, err)
			}
			emitSuccess(app.outputFormat, app.dependencies, commandResult{
				OK:        true,
				Operation: "build",
				ProjectID: plan.ProjectID,
				Profile:   plan.Profile.Config.ID,
				Output:    filepath.ToSlash(relativeManifestPath),
			})
			return nil
		},
	}
	command.Flags().StringVar(&profileOverride, "profile", "", "覆盖项目默认 Profile")
	return command
}

func (app *application) newFlashCommand() *cobra.Command {
	var profileOverride string
	var targetName string
	var dryRun bool
	command := &cobra.Command{
		Use:   "flash",
		Short: "将已验证固件写入 MCU target",
		Args:  noArguments("flash"),
		RunE: func(command *cobra.Command, _ []string) error {
			if targetName == "" {
				return invalidArguments("flash", fmt.Errorf("--target is required"))
			}
			plan, err := build.Resolve(build.OperationFlash, app.projectRoot, profileOverride, app.dependencies.Profiles)
			if err != nil {
				return resolutionFailure("flash", err)
			}
			verifiedOutput, err := output.LoadAndValidate(plan.OutputDirectory, plan.ProjectID, plan.Profile)
			if err != nil {
				return newCLIError("build_output_invalid", "flash", 1, err)
			}
			capability, exists := plan.Profile.Config.Targets[targetName]
			if !exists {
				return newCLIError("target_invalid", "flash", 1, fmt.Errorf("profile %q does not provide target %q", plan.Profile.Config.ID, targetName))
			}
			adapter, err := app.dependencies.FlashAdapters.Resolve(capability.Adapter)
			if err != nil {
				return newCLIError("target_invalid", "flash", 1, err)
			}
			flashResult, err := adapter.Flash(command.Context(), target.FlashRequest{
				BuildOutput: verifiedOutput,
				Profile:     plan.Profile,
				TargetName:  targetName,
				DryRun:      dryRun,
			})
			if err != nil {
				return newCLIError("target_failed", "flash", 1, err)
			}
			executed := flashResult.Executed
			emitSuccess(app.outputFormat, app.dependencies, commandResult{
				OK:        true,
				Operation: "flash",
				ProjectID: plan.ProjectID,
				Profile:   plan.Profile.Config.ID,
				Command:   flashResult.Command,
				Executed:  &executed,
			})
			return nil
		},
	}
	command.Flags().StringVar(&profileOverride, "profile", "", "覆盖项目默认 Profile")
	command.Flags().StringVar(&targetName, "target", "", "Profile 提供的 target capability")
	command.Flags().BoolVar(&dryRun, "dry-run", false, "只解析并显示 OpenOCD 命令")
	return command
}

func resolutionFailure(operation string, err error) *cliError {
	if errors.Is(err, build.ErrProject) {
		return newCLIError("project_invalid", operation, 1, err)
	}
	if errors.Is(err, build.ErrProfile) {
		return newCLIError("profile_invalid", operation, 1, err)
	}
	return newCLIError("profile_invalid", operation, 1, err)
}

func invalidArguments(operation string, err error) *cliError {
	return newCLIError("invalid_arguments", operation, 2, err)
}

func noArguments(operation string) cobra.PositionalArgs {
	return func(command *cobra.Command, arguments []string) error {
		if len(arguments) == 0 {
			return nil
		}
		return invalidArguments(operation, fmt.Errorf("unknown command %q for %q", arguments[0], command.CommandPath()))
	}
}

func newCLIError(code, operation string, exitCode int, err error) *cliError {
	return &cliError{code: code, operation: operation, exitCode: exitCode, err: err}
}

func emitSuccess(outputFormat string, dependencies Dependencies, result commandResult) {
	if outputFormat == "json" {
		_ = json.NewEncoder(dependencies.Stdout).Encode(result)
		return
	}
	switch result.Operation {
	case "init":
		fmt.Fprintf(dependencies.Stdout, "项目标识：%s\n", result.ProjectID)
	case "build":
		fmt.Fprintf(dependencies.Stdout, "构建完成：%s\n", result.Output)
	case "flash":
		if result.Executed != nil && *result.Executed {
			fmt.Fprintln(dependencies.Stdout, "OpenOCD 烧录完成")
		} else {
			fmt.Fprintf(dependencies.Stdout, "OpenOCD 命令（未执行）：%s\n", displayCommand(result.Command))
		}
	}
}

func emitFailure(outputFormat string, dependencies Dependencies, failure *cliError) {
	if outputFormat == "json" {
		_ = json.NewEncoder(dependencies.Stdout).Encode(failureResult{
			OK:        false,
			Operation: failure.operation,
			Error:     commandFailure{Code: failure.code, Message: failure.err.Error()},
		})
		return
	}
	fmt.Fprintf(dependencies.Stderr, "rm-relay: %s: %s\n", failure.code, failure.err)
}

func displayCommand(arguments []string) string {
	displayed := make([]string, len(arguments))
	for index, argument := range arguments {
		if argument == "" || strings.ContainsAny(argument, " \t\r\n\"") {
			displayed[index] = strconv.Quote(argument)
		} else {
			displayed[index] = argument
		}
	}
	return strings.Join(displayed, " ")
}
