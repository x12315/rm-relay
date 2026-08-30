package distribution

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/x12315/rm-relay/internal/execution/command"
)

func TestPackageSnapshotRejectsRepositoryOutput(t *testing.T) {
	packager := testPackager(t, &recordingRunner{})

	err := packager.PackageSnapshot(context.Background(), filepath.Join(packager.RepositoryRoot, "dist"))

	if err == nil || !strings.Contains(err.Error(), "outside repository") {
		t.Fatalf("PackageSnapshot() error = %v", err)
	}
}

func TestPackageSnapshotRequiresAbsoluteOutput(t *testing.T) {
	packager := testPackager(t, &recordingRunner{})

	err := packager.PackageSnapshot(context.Background(), "snapshot")

	if err == nil || !strings.Contains(err.Error(), "absolute") {
		t.Fatalf("PackageSnapshot() error = %v", err)
	}
}

func TestPackageSnapshotRunsGoReleaserInTemporaryClone(t *testing.T) {
	runner := &recordingRunner{}
	runner.onGoReleaser = func(directory string) error {
		return writeTestFile(filepath.Join(directory, "dist", "rm-relay_test.tar.gz"), "archive")
	}
	packager := testPackager(t, runner)
	output := filepath.Join(t.TempDir(), "snapshot")

	if err := packager.PackageSnapshot(context.Background(), output); err != nil {
		t.Fatal(err)
	}

	contents, err := os.ReadFile(filepath.Join(output, "rm-relay_test.tar.gz"))
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != "archive" {
		t.Fatalf("published archive = %q", contents)
	}
	for _, request := range runner.requests {
		if request.Name == "goreleaser" && testPathWithin(packager.RepositoryRoot, request.Directory) {
			t.Fatalf("GoReleaser ran inside source repository: %#v", request)
		}
	}
}

func TestPackageSnapshotDoesNotReplaceExistingOutput(t *testing.T) {
	packager := testPackager(t, &recordingRunner{})
	output := filepath.Join(t.TempDir(), "snapshot")
	if err := os.Mkdir(output, 0o755); err != nil {
		t.Fatal(err)
	}

	err := packager.PackageSnapshot(context.Background(), output)

	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("PackageSnapshot() error = %v", err)
	}
}

func TestPackageSnapshotRejectsUncommittedChanges(t *testing.T) {
	packager := testPackager(t, &recordingRunner{dirty: true})

	err := packager.PackageSnapshot(context.Background(), filepath.Join(t.TempDir(), "snapshot"))

	if err == nil || !strings.Contains(err.Error(), "uncommitted") {
		t.Fatalf("PackageSnapshot() error = %v", err)
	}
}

type recordingRunner struct {
	requests     []command.Request
	onGoReleaser func(string) error
	dirty        bool
}

func (runner *recordingRunner) Run(_ context.Context, request command.Request) (command.Result, error) {
	runner.requests = append(runner.requests, request)
	switch request.Name {
	case "git":
		if len(request.Arguments) > 0 && request.Arguments[0] == "status" {
			if runner.dirty {
				return command.Result{Stdout: " M changed.go\n"}, nil
			}
			return command.Result{}, nil
		}
		if len(request.Arguments) > 0 && request.Arguments[0] == "clone" {
			checkout := request.Arguments[len(request.Arguments)-1]
			return command.Result{}, os.MkdirAll(checkout, 0o755)
		}
		return command.Result{}, nil
	case "goreleaser":
		if runner.onGoReleaser == nil {
			return command.Result{}, fmt.Errorf("unexpected GoReleaser invocation")
		}
		return command.Result{}, runner.onGoReleaser(request.Directory)
	default:
		return command.Result{}, fmt.Errorf("unexpected command %q", request.Name)
	}
}

func testPackager(t *testing.T, runner command.Runner) Packager {
	t.Helper()
	repositoryRoot := filepath.Join(t.TempDir(), "repository")
	if err := os.Mkdir(repositoryRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	return Packager{Runner: runner, RepositoryRoot: repositoryRoot, GoReleaser: "goreleaser"}
}

func writeTestFile(path, contents string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(contents), 0o644)
}

func testPathWithin(parent, child string) bool {
	relative, err := filepath.Rel(parent, child)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}
