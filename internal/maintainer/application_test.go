package maintainer

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/x12315/rm-relay/internal/maintainer/experience"
)

func TestCLIPackageRequiresExplicitOutputDirectory(t *testing.T) {
	result := runTestApplication(t, []string{"cli", "package-snapshot"}, Actions{}, func(string) string { return "" })

	if result.exitCode != 1 || !strings.Contains(result.stderr, "RM_RELAY_CLI_OUTPUT_DIR") {
		t.Fatalf("result = %#v", result)
	}
}

func TestCLIPackagePassesConfiguredOutputDirectory(t *testing.T) {
	outputDirectory := filepath.Join(t.TempDir(), "snapshot")
	var actualOutput string
	actions := Actions{
		PackageSnapshot: func(_ context.Context, output string) error {
			actualOutput = output
			return nil
		},
	}
	result := runTestApplication(t, []string{"cli", "package-snapshot"}, actions, func(name string) string {
		if name == "RM_RELAY_CLI_OUTPUT_DIR" {
			return outputDirectory
		}
		return ""
	})

	if result.exitCode != 0 || actualOutput != outputDirectory || !strings.Contains(result.stdout, outputDirectory) {
		t.Fatalf("result = %#v output = %q", result, actualOutput)
	}
}

func TestExperiencePrepareReportsCandidateIdentity(t *testing.T) {
	actions := Actions{
		Prepare: func(context.Context) (experience.Prepared, error) {
			return experience.Prepared{Root: "/candidate", Revision: "abc123", CLIVersion: "snapshot", ImageID: "sha256:image", TemplateURL: "file:///template.git"}, nil
		},
	}
	result := runTestApplication(t, []string{"experience", "prepare"}, actions, func(string) string { return "" })

	if result.exitCode != 0 || !strings.Contains(result.stdout, "/candidate") || !strings.Contains(result.stdout, "abc123") {
		t.Fatalf("result = %#v", result)
	}
}

type applicationResult struct {
	exitCode int
	stdout   string
	stderr   string
}

func runTestApplication(t *testing.T, arguments []string, actions Actions, getenv func(string) string) applicationResult {
	t.Helper()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := Run(context.Background(), arguments, actions, Streams{Stdout: &stdout, Stderr: &stderr}, getenv)
	return applicationResult{exitCode: exitCode, stdout: stdout.String(), stderr: stderr.String()}
}
