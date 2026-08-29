// Package openocd adapts verified MCU Build Outputs to a development-machine OpenOCD process.
package openocd

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/x12315/rm-relay/toolkit/internal/commandexec"
	"github.com/x12315/rm-relay/toolkit/internal/miseexec"
	"github.com/x12315/rm-relay/toolkit/internal/profile"
	"github.com/x12315/rm-relay/toolkit/internal/target"
)

// Adapter invokes OpenOCD through the mise binary shipped with RM Relay.
type Adapter struct {
	Runner     commandexec.Runner
	MiseBinary string
	Progress   io.Writer
}

// Flash resolves a Profile capability and programs its verified firmware artifact.
func (adapter Adapter) Flash(ctx context.Context, request target.FlashRequest) (target.FlashResult, error) {
	capability, exists := request.Profile.Config.Targets[request.TargetName]
	if !exists {
		return target.FlashResult{}, fmt.Errorf("profile %q does not provide target %q", request.Profile.Config.ID, request.TargetName)
	}
	if capability.Adapter != "openocd" {
		return target.FlashResult{}, fmt.Errorf("target %q uses %q, not openocd", request.TargetName, capability.Adapter)
	}
	boardConfig, err := resolveTargetConfig(request.Profile, capability)
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
	openOCDCommand := []string{
		"openocd",
		"-f", boardConfig,
		"-c", "program {" + artifactPath + "} verify reset exit",
	}
	invocation := miseexec.ExecInvocation(
		[]string{filepath.Join(request.Profile.AssetsRoot, "mise", "core.toml")},
		openOCDCommand,
		string(os.PathListSeparator),
	)
	command := append([]string{adapter.MiseBinary}, invocation.Arguments...)
	result := target.FlashResult{Command: command, Executed: false}
	if request.DryRun {
		return result, nil
	}
	if adapter.Runner == nil {
		return target.FlashResult{}, fmt.Errorf("process runner is not configured")
	}
	processResult, err := adapter.Runner.Run(ctx, commandexec.Request{
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

func resolveTargetConfig(loadedProfile profile.Loaded, capability profile.Target) (string, error) {
	if loadedProfile.AssetsRoot == "" || !filepath.IsLocal(capability.Config) || filepath.Clean(capability.Config) == "." || strings.Contains(capability.Config, `\`) {
		return "", fmt.Errorf("target config %q is not a safe relative file", capability.Config)
	}
	assetsRoot, err := filepath.Abs(loadedProfile.AssetsRoot)
	if err != nil {
		return "", fmt.Errorf("resolve RM Relay assets: %w", err)
	}
	configPath := filepath.Join(assetsRoot, filepath.FromSlash(capability.Config))
	relativeToAssets, err := filepath.Rel(assetsRoot, configPath)
	if err != nil || !filepath.IsLocal(relativeToAssets) {
		return "", fmt.Errorf("target config %q escapes RM Relay assets", capability.Config)
	}
	fileInfo, err := os.Lstat(configPath)
	if err != nil {
		return "", fmt.Errorf("stat target config %q: %w", capability.Config, err)
	}
	if fileInfo.Mode()&os.ModeSymlink != 0 || !fileInfo.Mode().IsRegular() {
		return "", fmt.Errorf("target config %q must be a regular file", capability.Config)
	}
	return configPath, nil
}
