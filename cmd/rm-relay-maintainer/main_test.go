package main

import "testing"

func TestResolveShellUsesPlatformConvention(t *testing.T) {
	values := map[string]string{"SHELL": "/bin/zsh", "COMSPEC": `C:\Windows\System32\cmd.exe`}
	getenv := func(name string) string { return values[name] }

	if actual := resolveShell("darwin", getenv); actual != "/bin/zsh" {
		t.Fatalf("Darwin shell = %q", actual)
	}
	if actual := resolveShell("windows", getenv); actual != values["COMSPEC"] {
		t.Fatalf("Windows shell = %q", actual)
	}
}
