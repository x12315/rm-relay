// Package buildoutput owns the validated handoff from build backends to target adapters.
package buildoutput

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/x12315/rm-relay/toolkit/internal/executionplan"
	"github.com/x12315/rm-relay/toolkit/internal/profile"
)

const (
	// ManifestFileName is the deterministic metadata file stored beside Build Output artifacts.
	ManifestFileName = "rm-relay-output.json"

	currentSchemaVersion = 1
)

// Manifest records the identities and content hashes required by target adapters.
type Manifest struct {
	SchemaVersion      int        `json:"schema_version"`
	ProjectID          string     `json:"project_id"`
	ProfileID          string     `json:"profile_id"`
	ProfileDigest      string     `json:"profile_digest"`
	DevelopmentImage   string     `json:"development_image"`
	DevelopmentImageID string     `json:"development_image_id"`
	ProducerVersion    string     `json:"producer_version"`
	Artifacts          []Artifact `json:"artifacts"`
}

// Artifact identifies one deployable file by semantic role and content.
type Artifact struct {
	Role   string `json:"role"`
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

// Create validates declared outputs and atomically writes their deterministic manifest.
func Create(plan executionplan.Plan, imageID, producerVersion string) (Manifest, error) {
	if imageID == "" {
		return Manifest{}, fmt.Errorf("development image identity must not be empty")
	}
	if producerVersion == "" {
		return Manifest{}, fmt.Errorf("producer version must not be empty")
	}
	artifacts := make([]Artifact, 0, len(plan.Build.Outputs))
	for _, declaredOutput := range plan.Build.Outputs {
		artifact, err := inspectArtifact(plan.OutputDirectory, declaredOutput.Role, declaredOutput.Path)
		if err != nil {
			return Manifest{}, err
		}
		artifacts = append(artifacts, artifact)
	}
	sort.Slice(artifacts, func(left, right int) bool {
		return artifacts[left].Role < artifacts[right].Role
	})
	manifest := Manifest{
		SchemaVersion:      currentSchemaVersion,
		ProjectID:          plan.ProjectID,
		ProfileID:          plan.Profile.Config.ID,
		ProfileDigest:      plan.Profile.Digest,
		DevelopmentImage:   plan.Profile.Config.DevelopmentImage,
		DevelopmentImageID: imageID,
		ProducerVersion:    producerVersion,
		Artifacts:          artifacts,
	}
	if err := validateRequiredRoles(manifest, plan.Profile.Config.RequiredOutputRoles); err != nil {
		return Manifest{}, err
	}
	if err := writeManifest(plan.OutputDirectory, manifest); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

// LoadAndValidate rechecks manifest identity and every artifact before target use.
func LoadAndValidate(outputDirectory, projectID string, loadedProfile profile.Loaded) (Manifest, error) {
	manifestPath := filepath.Join(outputDirectory, ManifestFileName)
	manifestFile, err := os.Open(manifestPath)
	if err != nil {
		return Manifest{}, fmt.Errorf("open Build Output manifest: %w", err)
	}
	defer manifestFile.Close()

	decoder := json.NewDecoder(bufio.NewReader(manifestFile))
	decoder.DisallowUnknownFields()
	var manifest Manifest
	if err := decoder.Decode(&manifest); err != nil {
		return Manifest{}, fmt.Errorf("decode Build Output manifest: %w", err)
	}
	if err := ensureJSONEnd(decoder); err != nil {
		return Manifest{}, err
	}
	if manifest.SchemaVersion != currentSchemaVersion {
		return Manifest{}, fmt.Errorf("unsupported Build Output schema_version %d", manifest.SchemaVersion)
	}
	if manifest.ProjectID != projectID {
		return Manifest{}, fmt.Errorf("Build Output project identity %q does not match project %q", manifest.ProjectID, projectID)
	}
	if manifest.ProfileID != loadedProfile.Config.ID {
		return Manifest{}, fmt.Errorf("Build Output profile %q does not match profile %q", manifest.ProfileID, loadedProfile.Config.ID)
	}
	if manifest.ProfileDigest != loadedProfile.Digest {
		return Manifest{}, fmt.Errorf("Build Output profile digest does not match current profile")
	}
	if manifest.DevelopmentImage != loadedProfile.Config.DevelopmentImage {
		return Manifest{}, fmt.Errorf("Build Output development image does not match current profile")
	}
	if manifest.DevelopmentImageID == "" || manifest.ProducerVersion == "" {
		return Manifest{}, fmt.Errorf("Build Output manifest has incomplete producer identity")
	}
	if err := validateRequiredRoles(manifest, loadedProfile.Config.RequiredOutputRoles); err != nil {
		return Manifest{}, err
	}
	roles := make(map[string]struct{}, len(manifest.Artifacts))
	for _, artifact := range manifest.Artifacts {
		if artifact.Role == "" {
			return Manifest{}, fmt.Errorf("Build Output contains an empty artifact role")
		}
		if _, exists := roles[artifact.Role]; exists {
			return Manifest{}, fmt.Errorf("Build Output contains duplicate artifact role %q", artifact.Role)
		}
		roles[artifact.Role] = struct{}{}
		actual, err := inspectArtifact(outputDirectory, artifact.Role, artifact.Path)
		if err != nil {
			return Manifest{}, err
		}
		if actual.Size != artifact.Size {
			return Manifest{}, fmt.Errorf("artifact %q size changed: manifest=%d actual=%d", artifact.Role, artifact.Size, actual.Size)
		}
		if actual.SHA256 != artifact.SHA256 {
			return Manifest{}, fmt.Errorf("artifact %q SHA-256 changed: manifest=%s actual=%s", artifact.Role, artifact.SHA256, actual.SHA256)
		}
	}
	return manifest, nil
}

// ArtifactByRole returns the single artifact with the requested semantic role.
func (manifest Manifest) ArtifactByRole(role string) (Artifact, error) {
	var matches []Artifact
	for _, artifact := range manifest.Artifacts {
		if artifact.Role == role {
			matches = append(matches, artifact)
		}
	}
	switch len(matches) {
	case 0:
		return Artifact{}, fmt.Errorf("Build Output does not contain artifact role %q", role)
	case 1:
		return matches[0], nil
	default:
		return Artifact{}, fmt.Errorf("Build Output contains duplicate artifact role %q", role)
	}
}

func inspectArtifact(outputDirectory, role, relativePath string) (Artifact, error) {
	artifactPath, err := resolveArtifactPath(outputDirectory, relativePath)
	if err != nil {
		return Artifact{}, fmt.Errorf("artifact %q: %w", role, err)
	}
	fileInfo, err := os.Lstat(artifactPath)
	if err != nil {
		return Artifact{}, fmt.Errorf("artifact %q at %q: %w", role, relativePath, err)
	}
	if fileInfo.Mode()&os.ModeSymlink != 0 || !fileInfo.Mode().IsRegular() {
		return Artifact{}, fmt.Errorf("artifact %q at %q must be a regular file", role, relativePath)
	}
	hash, err := hashFile(artifactPath)
	if err != nil {
		return Artifact{}, fmt.Errorf("hash artifact %q: %w", role, err)
	}
	return Artifact{
		Role:   role,
		Path:   filepath.ToSlash(filepath.Clean(relativePath)),
		Size:   fileInfo.Size(),
		SHA256: hash,
	}, nil
}

func resolveArtifactPath(outputDirectory, relativePath string) (string, error) {
	if !filepath.IsLocal(relativePath) || filepath.Clean(relativePath) == "." || strings.Contains(relativePath, `\`) {
		return "", fmt.Errorf("path %q is not a safe relative file", relativePath)
	}
	outputRoot, err := filepath.Abs(outputDirectory)
	if err != nil {
		return "", fmt.Errorf("resolve output directory: %w", err)
	}
	artifactPath := filepath.Join(outputRoot, filepath.FromSlash(relativePath))
	relativeToRoot, err := filepath.Rel(outputRoot, artifactPath)
	if err != nil || !filepath.IsLocal(relativeToRoot) {
		return "", fmt.Errorf("path %q escapes the Build Output directory", relativePath)
	}
	return artifactPath, nil
}

func hashFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func validateRequiredRoles(manifest Manifest, requiredRoles []string) error {
	availableRoles := make(map[string]struct{}, len(manifest.Artifacts))
	for _, artifact := range manifest.Artifacts {
		availableRoles[artifact.Role] = struct{}{}
	}
	for _, role := range requiredRoles {
		if _, exists := availableRoles[role]; !exists {
			return fmt.Errorf("Build Output is missing required artifact role %q", role)
		}
	}
	return nil
}

func writeManifest(outputDirectory string, manifest Manifest) error {
	var content bytes.Buffer
	encoder := json.NewEncoder(&content)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(manifest); err != nil {
		return fmt.Errorf("encode Build Output manifest: %w", err)
	}
	temporaryFile, err := os.CreateTemp(outputDirectory, ".rm-relay-output-*")
	if err != nil {
		return fmt.Errorf("create temporary Build Output manifest: %w", err)
	}
	temporaryPath := temporaryFile.Name()
	defer os.Remove(temporaryPath)
	if err := temporaryFile.Chmod(0o644); err != nil {
		temporaryFile.Close()
		return fmt.Errorf("set Build Output manifest permissions: %w", err)
	}
	if _, err := temporaryFile.Write(content.Bytes()); err != nil {
		temporaryFile.Close()
		return fmt.Errorf("write Build Output manifest: %w", err)
	}
	if err := temporaryFile.Sync(); err != nil {
		temporaryFile.Close()
		return fmt.Errorf("sync Build Output manifest: %w", err)
	}
	if err := temporaryFile.Close(); err != nil {
		return fmt.Errorf("close Build Output manifest: %w", err)
	}
	if err := os.Rename(temporaryPath, filepath.Join(outputDirectory, ManifestFileName)); err != nil {
		return fmt.Errorf("replace Build Output manifest: %w", err)
	}
	return nil
}

func ensureJSONEnd(decoder *json.Decoder) error {
	var trailing any
	if err := decoder.Decode(&trailing); errors.Is(err, io.EOF) {
		return nil
	} else if err != nil {
		return fmt.Errorf("decode trailing Build Output data: %w", err)
	}
	return fmt.Errorf("Build Output manifest contains multiple JSON values")
}
