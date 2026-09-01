// Package environment defines the identity contract shared by environment images and consumers.
package environment

import (
	"fmt"
	"strings"

	"github.com/BurntSushi/toml"
)

const (
	// IdentityPath is the immutable in-image location exported during environment verification.
	IdentityPath          = "/opt/rm-relay/environment/identity.toml"
	identitySchemaVersion = 1
)

// Identity identifies one compatible RM Relay development environment.
type Identity struct {
	SchemaVersion int    `toml:"schema_version"`
	ID            string `toml:"id"`
}

// DigestReference is an OCI image name paired with an immutable SHA-256 manifest digest.
type DigestReference struct {
	ImageName string
	Digest    string
}

// ParseIdentity strictly decodes and validates an environment image identity document.
func ParseIdentity(content []byte) (Identity, error) {
	var identity Identity
	metadata, err := toml.Decode(string(content), &identity)
	if err != nil {
		return Identity{}, fmt.Errorf("decode environment identity: %w", err)
	}
	if undecoded := metadata.Undecoded(); len(undecoded) > 0 {
		return Identity{}, fmt.Errorf("environment identity contains unknown key %q", undecoded[0].String())
	}
	if identity.SchemaVersion != identitySchemaVersion {
		return Identity{}, fmt.Errorf("unsupported environment identity schema_version %d", identity.SchemaVersion)
	}
	if !IsIdentifier(identity.ID) {
		return Identity{}, fmt.Errorf("environment identity ID %q is invalid", identity.ID)
	}
	return identity, nil
}

// ParseDigestReference requires an OCI image reference pinned to a lowercase SHA-256 digest.
func ParseDigestReference(value string) (DigestReference, error) {
	const marker = "@sha256:"
	if strings.Count(value, marker) != 1 || strings.ContainsAny(value, " \t\r\n\"") {
		return DigestReference{}, fmt.Errorf("environment reference must use image@sha256:<64 lowercase hex>")
	}
	markerIndex := strings.LastIndex(value, marker)
	if markerIndex <= 0 || strings.Contains(value[:markerIndex], "@") {
		return DigestReference{}, fmt.Errorf("environment reference must use image@sha256:<64 lowercase hex>")
	}
	digestHex := value[markerIndex+len(marker):]
	if len(digestHex) != 64 {
		return DigestReference{}, fmt.Errorf("environment reference must use image@sha256:<64 lowercase hex>")
	}
	for _, character := range digestHex {
		if (character < '0' || character > '9') && (character < 'a' || character > 'f') {
			return DigestReference{}, fmt.Errorf("environment reference must use image@sha256:<64 lowercase hex>")
		}
	}
	return DigestReference{ImageName: value[:markerIndex], Digest: "sha256:" + digestHex}, nil
}

// IsIdentifier reports whether value is safe as a stable environment catalog key.
func IsIdentifier(value string) bool {
	if value == "" {
		return false
	}
	for index, character := range value {
		if (character >= 'a' && character <= 'z') ||
			(character >= 'A' && character <= 'Z') ||
			(character >= '0' && character <= '9') ||
			(index > 0 && (character == '-' || character == '_' || character == '.')) {
			continue
		}
		return false
	}
	return true
}
