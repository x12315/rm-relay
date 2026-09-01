package environment

import (
	"strings"
	"testing"
)

func TestParseIdentityAcceptsTheCurrentSchema(t *testing.T) {
	identity, err := ParseIdentity([]byte("schema_version = 1\nid = \"embedded-development\"\n"))
	if err != nil {
		t.Fatal(err)
	}
	if identity.ID != "embedded-development" {
		t.Fatalf("identity ID = %q", identity.ID)
	}
}

func TestParseIdentityRejectsUnknownFields(t *testing.T) {
	_, err := ParseIdentity([]byte("schema_version = 1\nid = \"embedded-development\"\nextra = true\n"))
	if err == nil || !strings.Contains(err.Error(), "unknown key") {
		t.Fatalf("ParseIdentity() error = %v", err)
	}
}

func TestParseIdentityRejectsUnsupportedSchema(t *testing.T) {
	_, err := ParseIdentity([]byte("schema_version = 2\nid = \"embedded-development\"\n"))
	if err == nil || !strings.Contains(err.Error(), "schema_version") {
		t.Fatalf("ParseIdentity() error = %v", err)
	}
}

func TestParseDigestReferenceReturnsRepositoryAndDigest(t *testing.T) {
	digest := "sha256:" + strings.Repeat("a", 64)
	reference, err := ParseDigestReference("registry.example.org/rm-relay/embedded:v0.1.0@" + digest)
	if err != nil {
		t.Fatal(err)
	}
	if reference.ImageName != "registry.example.org/rm-relay/embedded:v0.1.0" || reference.Digest != digest {
		t.Fatalf("reference = %#v", reference)
	}
}

func TestParseDigestReferenceRejectsMutableOrMalformedReferences(t *testing.T) {
	for _, reference := range []string{
		"registry.example.org/image:latest",
		"registry.example.org/image@sha256:" + strings.Repeat("A", 64),
		"registry.example.org/image@sha256:" + strings.Repeat("a", 63),
		"registry.example.org/image@sha256:" + strings.Repeat("a", 64) + "@sha256:" + strings.Repeat("b", 64),
		"registry.example.org/image with-space@sha256:" + strings.Repeat("a", 64),
	} {
		if _, err := ParseDigestReference(reference); err == nil {
			t.Fatalf("malformed reference accepted: %q", reference)
		}
	}
}
