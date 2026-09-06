// Package mise builds controlled mise invocations without executing processes.
package mise

import (
	_ "embed"
	"strings"

	"github.com/x12315/rm-relay/internal/execution/resourcecache"
)

const ContainerBaseConfig = "/opt/rm-relay/execution/mise/base.mise.toml"

//go:embed base.mise.toml
var baseConfig []byte

// Invocation contains mise arguments and the environment values RM Relay owns.
type Invocation struct {
	Arguments   []string
	Environment map[string]string
}

// MaterializeBaseConfig returns the host path to RM Relay's isolated mise base configuration.
func MaterializeBaseConfig(store resourcecache.Store) (string, error) {
	return store.Materialize("execution/mise", "base.mise.toml", baseConfig)
}

// TaskInvocation creates a locked task invocation isolated from discovered mise configs.
func TaskInvocation(configPaths []string, task, pathSeparator string) Invocation {
	return Invocation{
		Arguments: []string{"--locked", "run", task},
		Environment: map[string]string{
			"MISE_OVERRIDE_CONFIG_FILENAMES": strings.Join(configPaths, pathSeparator),
			"MISE_TASK_RUN_AUTO_INSTALL":     "false",
		},
	}
}

// ExecInvocation creates a mise exec command while preserving native argument boundaries.
func ExecInvocation(configPaths, command []string, pathSeparator string) Invocation {
	arguments := make([]string, 0, len(command)+2)
	arguments = append(arguments, "exec", "--")
	arguments = append(arguments, command...)
	return Invocation{
		Arguments: arguments,
		Environment: map[string]string{
			"MISE_OVERRIDE_CONFIG_FILENAMES": strings.Join(configPaths, pathSeparator),
			"MISE_TASK_RUN_AUTO_INSTALL":     "false",
		},
	}
}
