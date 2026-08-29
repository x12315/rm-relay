// Package cli composes RM Relay contracts, backends and target adapters into the public command line.
package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/x12315/rm-relay/toolkit/internal/backend/localcontainer"
	"github.com/x12315/rm-relay/toolkit/internal/buildoutput"
	"github.com/x12315/rm-relay/toolkit/internal/commandexec"
	"github.com/x12315/rm-relay/toolkit/internal/executionplan"
	"github.com/x12315/rm-relay/toolkit/internal/project"
	"github.com/x12315/rm-relay/toolkit/internal/target"
	openocdtarget "github.com/x12315/rm-relay/toolkit/internal/target/openocd"
	"github.com/x12315/rm-relay/toolkit/internal/workspacebuild"
)

// Dependencies contains process and distribution boundaries supplied by the executable.
type Dependencies struct {
	Runner          commandexec.Runner
	HomeDirectory   string
	MiseBinary      string
	ProducerVersion string
	Stdout          io.Writer
	Stderr          io.Writer
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
	result, outputFormat, failure := execute(ctx, arguments, dependencies)
	if failure != nil {
		emitFailure(outputFormat, dependencies, failure)
		return failure.exitCode
	}
	emitSuccess(outputFormat, dependencies, result)
	return 0
}

func execute(ctx context.Context, arguments []string, dependencies Dependencies) (commandResult, string, *cliError) {
	globalFlags := flag.NewFlagSet("rm-relay", flag.ContinueOnError)
	globalFlags.SetOutput(io.Discard)
	projectRoot := globalFlags.String("project", ".", "project root")
	outputFormat := globalFlags.String("format", "human", "human or json")
	if err := globalFlags.Parse(arguments); err != nil {
		return commandResult{}, *outputFormat, invalidArguments("", err)
	}
	if *outputFormat != "human" && *outputFormat != "json" {
		return commandResult{}, *outputFormat, invalidArguments("", fmt.Errorf("unsupported output format %q", *outputFormat))
	}
	remaining := globalFlags.Args()
	if len(remaining) == 0 {
		return commandResult{}, *outputFormat, invalidArguments("", fmt.Errorf("a command is required: init, build or flash"))
	}
	command := remaining[0]
	commandArguments := remaining[1:]
	switch command {
	case "init":
		result, failure := runInit(*projectRoot, commandArguments)
		return result, *outputFormat, failure
	case "build":
		result, failure := runBuild(ctx, *projectRoot, commandArguments, dependencies)
		return result, *outputFormat, failure
	case "flash":
		result, failure := runFlash(ctx, *projectRoot, commandArguments, dependencies)
		return result, *outputFormat, failure
	default:
		return commandResult{}, *outputFormat, invalidArguments(command, fmt.Errorf("unknown command %q", command))
	}
}

func runInit(projectRoot string, arguments []string) (commandResult, *cliError) {
	if len(arguments) != 0 {
		return commandResult{}, invalidArguments("init", fmt.Errorf("init does not accept positional arguments"))
	}
	projectID, err := project.Initialize(projectRoot)
	if err != nil {
		return commandResult{}, newCLIError("project_invalid", "init", 1, err)
	}
	return commandResult{OK: true, Operation: "init", ProjectID: projectID}, nil
}

func runBuild(ctx context.Context, projectRoot string, arguments []string, dependencies Dependencies) (commandResult, *cliError) {
	flags := flag.NewFlagSet("build", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	profileOverride := flags.String("profile", "", "RM Relay Profile ID")
	if err := flags.Parse(arguments); err != nil {
		return commandResult{}, invalidArguments("build", err)
	}
	if len(flags.Args()) != 0 {
		return commandResult{}, invalidArguments("build", fmt.Errorf("build does not accept positional arguments"))
	}
	assetsRoot := filepath.Join(dependencies.HomeDirectory, "share", "rm-relay")
	plan, err := executionplan.Resolve(executionplan.OperationBuild, projectRoot, assetsRoot, *profileOverride)
	if err != nil {
		return commandResult{}, resolutionFailure("build", err)
	}
	service := workspacebuild.Service{
		Backend: localcontainer.Backend{
			Runner:   dependencies.Runner,
			Progress: dependencies.Stderr,
		},
		ProducerVersion: dependencies.ProducerVersion,
	}
	if _, err := service.Execute(ctx, plan); err != nil {
		return commandResult{}, newCLIError("build_failed", "build", 1, err)
	}
	manifestPath := filepath.Join(plan.OutputDirectory, buildoutput.ManifestFileName)
	relativeManifestPath, err := filepath.Rel(plan.ProjectRoot, manifestPath)
	if err != nil {
		return commandResult{}, newCLIError("build_output_invalid", "build", 1, err)
	}
	return commandResult{
		OK:        true,
		Operation: "build",
		ProjectID: plan.ProjectID,
		Profile:   plan.Profile.Config.ID,
		Output:    filepath.ToSlash(relativeManifestPath),
	}, nil
}

func runFlash(ctx context.Context, projectRoot string, arguments []string, dependencies Dependencies) (commandResult, *cliError) {
	flags := flag.NewFlagSet("flash", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	profileOverride := flags.String("profile", "", "RM Relay Profile ID")
	targetName := flags.String("target", "", "Profile target capability")
	dryRun := flags.Bool("dry-run", false, "resolve without running OpenOCD")
	if err := flags.Parse(arguments); err != nil {
		return commandResult{}, invalidArguments("flash", err)
	}
	if len(flags.Args()) != 0 {
		return commandResult{}, invalidArguments("flash", fmt.Errorf("flash does not accept positional arguments"))
	}
	if *targetName == "" {
		return commandResult{}, invalidArguments("flash", fmt.Errorf("--target is required"))
	}
	assetsRoot := filepath.Join(dependencies.HomeDirectory, "share", "rm-relay")
	plan, err := executionplan.Resolve(executionplan.OperationFlash, projectRoot, assetsRoot, *profileOverride)
	if err != nil {
		return commandResult{}, resolutionFailure("flash", err)
	}
	verifiedOutput, err := buildoutput.LoadAndValidate(plan.OutputDirectory, plan.ProjectID, plan.Profile)
	if err != nil {
		return commandResult{}, newCLIError("build_output_invalid", "flash", 1, err)
	}
	capability, exists := plan.Profile.Config.Targets[*targetName]
	if !exists || capability.Adapter != "openocd" {
		return commandResult{}, newCLIError("target_invalid", "flash", 1, fmt.Errorf("profile %q does not provide OpenOCD target %q", plan.Profile.Config.ID, *targetName))
	}
	adapter := openocdtarget.Adapter{
		Runner:     dependencies.Runner,
		MiseBinary: dependencies.MiseBinary,
		Progress:   dependencies.Stderr,
	}
	flashResult, err := adapter.Flash(ctx, target.FlashRequest{
		BuildOutput: verifiedOutput,
		Profile:     plan.Profile,
		TargetName:  *targetName,
		DryRun:      *dryRun,
	})
	if err != nil {
		return commandResult{}, newCLIError("target_failed", "flash", 1, err)
	}
	executed := flashResult.Executed
	return commandResult{
		OK:        true,
		Operation: "flash",
		ProjectID: plan.ProjectID,
		Profile:   plan.Profile.Config.ID,
		Command:   flashResult.Command,
		Executed:  &executed,
	}, nil
}

func resolutionFailure(operation string, err error) *cliError {
	if errors.Is(err, executionplan.ErrProject) {
		return newCLIError("project_invalid", operation, 1, err)
	}
	if errors.Is(err, executionplan.ErrProfile) {
		return newCLIError("profile_invalid", operation, 1, err)
	}
	return newCLIError("profile_invalid", operation, 1, err)
}

func invalidArguments(operation string, err error) *cliError {
	return newCLIError("invalid_arguments", operation, 2, err)
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
			Error: commandFailure{
				Code:    failure.code,
				Message: failure.err.Error(),
			},
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
