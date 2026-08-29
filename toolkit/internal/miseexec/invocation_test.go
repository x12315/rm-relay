package miseexec

import (
	"reflect"
	"testing"
)

func TestTaskInvocationIsLockedAndDisablesAutoInstall(t *testing.T) {
	invocation := TaskInvocation([]string{"/core.toml", "/profile.toml", "/project.toml"}, "build:firmware", ":")

	if !reflect.DeepEqual(invocation.Arguments, []string{"--locked", "run", "build:firmware"}) {
		t.Fatalf("Arguments = %v", invocation.Arguments)
	}
	if invocation.Environment["MISE_TASK_RUN_AUTO_INSTALL"] != "false" {
		t.Fatalf("MISE_TASK_RUN_AUTO_INSTALL = %q", invocation.Environment["MISE_TASK_RUN_AUTO_INSTALL"])
	}
}

func TestTaskInvocationUsesOnlyExplicitConfigFiles(t *testing.T) {
	invocation := TaskInvocation([]string{"/core.toml", "/profile.toml", "/project.toml"}, "build:firmware", ":")

	if got := invocation.Environment["MISE_OVERRIDE_CONFIG_FILENAMES"]; got != "/core.toml:/profile.toml:/project.toml" {
		t.Fatalf("MISE_OVERRIDE_CONFIG_FILENAMES = %q", got)
	}
	if len(invocation.Environment) != 2 {
		t.Fatalf("Environment = %v, want only controlled mise variables", invocation.Environment)
	}
}

func TestExecInvocationPreservesArgumentBoundaries(t *testing.T) {
	command := []string{"openocd", "-f", "/path with spaces/board.cfg", "-c", "program {/firmware path/app.elf} verify reset exit"}

	invocation := ExecInvocation(command)
	want := append([]string{"exec", "--"}, command...)
	if !reflect.DeepEqual(invocation.Arguments, want) {
		t.Fatalf("Arguments = %#v, want %#v", invocation.Arguments, want)
	}
}
