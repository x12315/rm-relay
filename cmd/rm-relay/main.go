package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"

	"github.com/x12315/rm-relay/internal/build"
	"github.com/x12315/rm-relay/internal/build/backend/localcontainer"
	"github.com/x12315/rm-relay/internal/build/cmake"
	"github.com/x12315/rm-relay/internal/cli"
	"github.com/x12315/rm-relay/internal/execution/command"
	"github.com/x12315/rm-relay/internal/execution/resourcecache"
	"github.com/x12315/rm-relay/internal/profile"
	"github.com/x12315/rm-relay/internal/target"
	"github.com/x12315/rm-relay/internal/target/openocd"
)

var version = "development"

func main() {
	miseBinary := resolveMiseBinary(os.Getenv, runtime.GOOS)
	cacheRoot, err := resolveCacheDirectory()
	if err != nil {
		fmt.Fprintf(os.Stderr, "rm-relay: environment_invalid: %s\n", err)
		os.Exit(1)
	}
	workflows, err := build.NewWorkflowCatalog(cmake.Workflow{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "rm-relay: distribution_invalid: %s\n", err)
		os.Exit(1)
	}
	resourceStore := resourcecache.Store{Root: cacheRoot}
	processRunner := command.OSRunner{}
	buildBackends, err := build.NewBackendCatalog(localcontainer.Backend{
		Runner:         processRunner,
		Workflows:      workflows,
		CacheDirectory: filepath.Join(cacheRoot, "build", localcontainer.ID),
		Progress:       os.Stderr,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "rm-relay: distribution_invalid: %s\n", err)
		os.Exit(1)
	}
	flashAdapters, err := target.NewFlashAdapterCatalog(openocd.Adapter{
		Runner:        processRunner,
		MiseBinary:    miseBinary,
		ResourceCache: resourceStore,
		Boards:        openocd.BuiltinBoardCatalog(resourceStore),
		Progress:      os.Stderr,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "rm-relay: distribution_invalid: %s\n", err)
		os.Exit(1)
	}
	commandContext, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt)
	exitCode := cli.Run(commandContext, os.Args[1:], cli.Dependencies{
		Profiles:        profile.BuiltinCatalog(),
		BuildBackends:   buildBackends,
		DefaultBackend:  localcontainer.ID,
		FlashAdapters:   flashAdapters,
		ProducerVersion: version,
		Stdout:          os.Stdout,
		Stderr:          os.Stderr,
	})
	stopSignals()
	os.Exit(exitCode)
}

func resolveMiseBinary(getenv func(string) string, goos string) string {
	if configuredBinary := getenv("RM_RELAY_MISE_BIN"); configuredBinary != "" {
		return configuredBinary
	}
	if goos == "windows" {
		return "mise.exe"
	}
	return "mise"
}

func resolveCacheDirectory() (string, error) {
	if configuredCache := os.Getenv("RM_RELAY_CACHE_DIR"); configuredCache != "" {
		absoluteCache, err := filepath.Abs(configuredCache)
		if err != nil {
			return "", fmt.Errorf("resolve RM_RELAY_CACHE_DIR: %w", err)
		}
		return absoluteCache, nil
	}
	userCache, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("locate user cache directory: %w", err)
	}
	return filepath.Join(userCache, "rm-relay"), nil
}
