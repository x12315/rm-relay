// Package openocd adapts verified MCU Build Outputs to a development-machine OpenOCD process.
package openocd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/x12315/rm-relay/internal/execution/command"
	"github.com/x12315/rm-relay/internal/execution/mise"
	"github.com/x12315/rm-relay/internal/execution/resourcecache"
	"github.com/x12315/rm-relay/internal/target"
)

const adapterID = "openocd"

// Adapter invokes OpenOCD through the host mise binary selected by RM Relay.
type Adapter struct {
	Runner        command.Runner
	MiseBinary    string
	ResourceCache resourcecache.Store
	Boards        BoardCatalog
	Progress      io.Writer
}

// ID returns the semantic adapter identifier used by Profile target capabilities.
func (Adapter) ID() string {
	return adapterID
}

// Flash resolves a Profile capability and programs its verified firmware artifact.
func (adapter Adapter) Flash(ctx context.Context, request target.FlashRequest) (target.FlashResult, error) {
	capability, exists := request.Profile.Config.Targets[request.TargetName]
	if !exists {
		return target.FlashResult{}, fmt.Errorf("profile %q does not provide target %q", request.Profile.Config.ID, request.TargetName)
	}
	if capability.Adapter != adapterID {
		return target.FlashResult{}, fmt.Errorf("target %q uses %q, not %s", request.TargetName, capability.Adapter, adapterID)
	}
	boardConfig, err := adapter.Boards.Resolve(capability.Board)
	if err != nil {
		return target.FlashResult{}, err
	}
	artifactPath, err := request.BuildOutput.ArtifactPathByRole(capability.ArtifactRole)
	if err != nil {
		return target.FlashResult{}, err
	}
	if adapter.MiseBinary == "" {
		return target.FlashResult{}, fmt.Errorf("RM Relay mise binary is not configured")
	}
	miseConfigPath, err := mise.MaterializeBaseConfig(adapter.ResourceCache)
	if err != nil {
		return target.FlashResult{}, fmt.Errorf("materialize RM Relay mise config: %w", err)
	}
	openOCDCommand := []string{
		"openocd",
		"-f", boardConfig,
		"-c", "program {" + artifactPath + "} verify reset exit",
	}
	invocation := mise.ExecInvocation(
		[]string{miseConfigPath},
		openOCDCommand,
		string(os.PathListSeparator),
	)
	fullCommand := append([]string{adapter.MiseBinary}, invocation.Arguments...)
	result := target.FlashResult{Command: fullCommand, Executed: false}
	if request.DryRun {
		return result, nil
	}
	if adapter.Runner == nil {
		return target.FlashResult{}, fmt.Errorf("process runner is not configured")
	}
	processResult, err := adapter.Runner.Run(ctx, command.Request{
		Name:        adapter.MiseBinary,
		Arguments:   invocation.Arguments,
		Environment: invocation.Environment,
		Stdout:      adapter.Progress,
		Stderr:      adapter.Progress,
	})
	if err != nil {
		details := strings.TrimSpace(processResult.Stderr)
		if details == "" {
			return target.FlashResult{}, fmt.Errorf("run OpenOCD: %w", err)
		}
		return target.FlashResult{}, fmt.Errorf("run OpenOCD: %w: %s", err, details)
	}
	result.Executed = true
	return result, nil
}
