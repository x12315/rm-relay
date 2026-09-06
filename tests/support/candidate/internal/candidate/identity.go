package candidate

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/x12315/rm-relay/internal/execution/command"
)

func layoutWithRoot(layout Layout, root string) Layout {
	binaryDirectory := filepath.Join(root, "bin")
	binaryName := filepath.Base(layout.BinaryPath)
	layout.Root = root
	layout.StatePath = filepath.Join(root, "state.json")
	layout.BinaryDirectory = binaryDirectory
	layout.BinaryPath = filepath.Join(binaryDirectory, binaryName)
	layout.ConfigDirectory = filepath.Join(root, "config")
	layout.TemplateOrigin = filepath.Join(root, "template.git")
	layout.Workspace = filepath.Join(root, "workspace")
	layout.Logs = filepath.Join(root, "logs")
	return layout
}

func templateURL(path string) string {
	return (&url.URL{Scheme: "file", Path: filepath.ToSlash(path)}).String()
}

func fileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open candidate CLI: %w", err)
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("hash candidate CLI: %w", err)
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func oneIdentity(name, output string) (string, error) {
	fields := strings.Fields(output)
	if len(fields) != 1 {
		return "", fmt.Errorf("%s returned %d identities", name, len(fields))
	}
	return fields[0], nil
}

func candidateProcessFailure(action string, result command.Result, processError error) error {
	details := strings.TrimSpace(result.Stderr)
	if details == "" {
		return fmt.Errorf("%s: %w", action, processError)
	}
	return fmt.Errorf("%s: %w: %s", action, processError, details)
}
