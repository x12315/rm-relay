// Package miseexec builds controlled mise invocations without executing processes.
package miseexec

import "strings"

// Invocation contains mise arguments and the environment values RM Relay owns.
type Invocation struct {
	Arguments   []string
	Environment map[string]string
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
func ExecInvocation(command []string) Invocation {
	arguments := make([]string, 0, len(command)+2)
	arguments = append(arguments, "exec", "--")
	arguments = append(arguments, command...)
	return Invocation{Arguments: arguments, Environment: map[string]string{}}
}
