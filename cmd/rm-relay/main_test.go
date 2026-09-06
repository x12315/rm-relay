package main

import "testing"

func TestResolveMiseBinaryUsesPathName(t *testing.T) {
	if got := resolveMiseBinary(func(string) string { return "" }, "darwin"); got != "mise" {
		t.Fatalf("resolveMiseBinary() = %q, want mise", got)
	}
}

func TestResolveMiseBinaryUsesWindowsExecutableName(t *testing.T) {
	if got := resolveMiseBinary(func(string) string { return "" }, "windows"); got != "mise.exe" {
		t.Fatalf("resolveMiseBinary() = %q, want mise.exe", got)
	}
}

func TestResolveMiseBinaryHonorsExplicitOverride(t *testing.T) {
	getenv := func(name string) string {
		if name == "RM_RELAY_MISE_BIN" {
			return "/opt/tools/mise"
		}
		return ""
	}
	if got := resolveMiseBinary(getenv, "linux"); got != "/opt/tools/mise" {
		t.Fatalf("resolveMiseBinary() = %q, want /opt/tools/mise", got)
	}
}
