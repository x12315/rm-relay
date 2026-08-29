package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/x12315/rm-relay/toolkit/internal/cli"
	"github.com/x12315/rm-relay/toolkit/internal/commandexec"
)

var version = "development"

func main() {
	homeDirectory, err := distributionHome()
	if err != nil {
		fmt.Fprintf(os.Stderr, "rm-relay: distribution_invalid: %s\n", err)
		os.Exit(1)
	}
	miseBinary := os.Getenv("RM_RELAY_MISE_BIN")
	if miseBinary == "" {
		miseName := "mise"
		if runtime.GOOS == "windows" {
			miseName += ".exe"
		}
		miseBinary = filepath.Join(homeDirectory, "bin", miseName)
	}
	exitCode := cli.Run(context.Background(), os.Args[1:], cli.Dependencies{
		Runner:          commandexec.OSRunner{},
		HomeDirectory:   homeDirectory,
		MiseBinary:      miseBinary,
		ProducerVersion: version,
		Stdout:          os.Stdout,
		Stderr:          os.Stderr,
	})
	os.Exit(exitCode)
}

func distributionHome() (string, error) {
	if configuredHome := os.Getenv("RM_RELAY_HOME"); configuredHome != "" {
		absoluteHome, err := filepath.Abs(configuredHome)
		if err != nil {
			return "", fmt.Errorf("resolve RM_RELAY_HOME: %w", err)
		}
		return absoluteHome, nil
	}
	executable, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("locate rm-relay executable: %w", err)
	}
	executable, err = filepath.EvalSymlinks(executable)
	if err != nil {
		return "", fmt.Errorf("resolve rm-relay executable: %w", err)
	}
	return filepath.Dir(filepath.Dir(executable)), nil
}
