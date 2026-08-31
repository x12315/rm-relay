package buildx

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/x12315/rm-relay/internal/execution/command"
)

type recordingRunner struct{ requests []command.Request }

func (runner *recordingRunner) Run(_ context.Context, request command.Request) (command.Result, error) {
	runner.requests = append(runner.requests, request)
	return command.Result{}, nil
}

func TestCreateRemoteBuildsExactMTLSArguments(t *testing.T) {
	root := t.TempDir()
	paths := []string{filepath.Join(root, "ca.pem"), filepath.Join(root, "user.pem"), filepath.Join(root, "user-key.pem")}
	for _, path := range paths {
		if err := os.WriteFile(path, []byte("test"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	runner := &recordingRunner{}
	request := CreateRemoteRequest{Name: "rm-relay-team", Endpoint: "tcp://build.example.org:1234", CAPath: paths[0], CertificatePath: paths[1], KeyPath: paths[2], ServerName: "build.example.org"}
	if err := (CLI{Runner: runner}).CreateRemote(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	want := []string{"buildx", "create", "--name", "rm-relay-team", "--driver", "remote", "--driver-opt", "cacert=" + paths[0] + ",cert=" + paths[1] + ",key=" + paths[2] + ",servername=build.example.org", "tcp://build.example.org:1234"}
	if !reflect.DeepEqual(runner.requests[0].Arguments, want) {
		t.Fatalf("arguments = %#v, want %#v", runner.requests[0].Arguments, want)
	}
}

func TestCreateRemoteRejectsNonTLSEscapeCharacters(t *testing.T) {
	err := (CLI{Runner: &recordingRunner{}}).CreateRemote(context.Background(), CreateRemoteRequest{Name: "team", Endpoint: "tcp://host:1234", CAPath: "/tmp/ca.pem,cert=bad", CertificatePath: "/tmp/cert", KeyPath: "/tmp/key", ServerName: "host"})
	if err == nil {
		t.Fatal("CreateRemote() accepted option injection")
	}
}
