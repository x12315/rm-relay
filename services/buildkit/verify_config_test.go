package buildkit_test

import (
	"os"
	"strings"
	"testing"
)

func TestComposeKeepsBuildKitRootlessPinnedAndMTLSOnly(t *testing.T) {
	content, err := os.ReadFile("compose.yaml")
	if err != nil {
		t.Fatal(err)
	}
	compose := string(content)
	for _, required := range []string{
		"moby/buildkit:v0.32.2-rootless@sha256:504731e577c20559c00f968f33219f30115e70be29ab96728d1d06e963fc494b",
		"--tlscacert=/run/rm-relay-tls/ca.pem", "--tlscert=/run/rm-relay-tls/cert.pem", "--tlskey=/run/rm-relay-tls/key.pem",
		"/run/rm-relay-tls:ro", "buildkit-state:", "systempaths=unconfined",
	} {
		if !strings.Contains(compose, required) {
			t.Fatalf("compose.yaml missing %q", required)
		}
	}
	for _, forbidden := range []string{"/var/run/docker.sock", "privileged: true", "--oci-worker-no-process-sandbox"} {
		if strings.Contains(compose, forbidden) {
			t.Fatalf("compose.yaml contains forbidden %q", forbidden)
		}
	}
	if count := strings.Count(compose, "--addr="); count != 1 {
		t.Fatalf("compose.yaml defines %d BuildKit listeners, want exactly one mTLS listener", count)
	}
	if !strings.Contains(compose, "--addr=tcp://0.0.0.0:1234") {
		t.Fatal("compose.yaml does not define the expected TCP listener")
	}
}
