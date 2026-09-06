package docker

import (
	"context"
	"reflect"
	"testing"

	"github.com/x12315/rm-relay/internal/execution/command"
)

type recordingRunner struct {
	requests []command.Request
	result   command.Result
}

func (runner *recordingRunner) Run(_ context.Context, request command.Request) (command.Result, error) {
	runner.requests = append(runner.requests, request)
	return runner.result, nil
}

func TestRunPreservesNativeArgumentBoundaries(t *testing.T) {
	runner := &recordingRunner{}
	err := (CLI{Runner: runner}).Run(context.Background(), RunRequest{Image: "image:test", Volumes: []string{"/host path:/workspace"}, Workdir: "/workspace", Environment: map[string]string{"B": "two words", "A": "one"}, Command: []string{"tool", "argument with spaces"}})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"run", "--rm", "--volume", "/host path:/workspace", "--workdir", "/workspace", "--env", "A=one", "--env", "B=two words", "image:test", "tool", "argument with spaces"}
	if !reflect.DeepEqual(runner.requests[0].Arguments, want) {
		t.Fatalf("arguments = %#v, want %#v", runner.requests[0].Arguments, want)
	}
}

func TestInspectImageRequiresOneIdentity(t *testing.T) {
	runner := &recordingRunner{result: command.Result{Stdout: "sha256:one\nsha256:two\n"}}
	if _, err := (CLI{Runner: runner}).InspectImage(context.Background(), "image:test"); err == nil {
		t.Fatal("InspectImage() accepted multiple identities")
	}
}
