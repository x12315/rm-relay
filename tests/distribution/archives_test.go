//go:build distribution

package distribution

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/x12315/rm-relay/internal/execution/command"
)

var expectedArchives = map[string][]string{
	"darwin_amd64":  {"rm-relay", "LICENSE"},
	"darwin_arm64":  {"rm-relay", "LICENSE"},
	"linux_amd64":   {"rm-relay", "LICENSE"},
	"linux_arm64":   {"rm-relay", "LICENSE"},
	"windows_amd64": {"rm-relay.exe", "LICENSE"},
	"windows_arm64": {"rm-relay.exe", "LICENSE"},
}

func TestSnapshotArchivesContainOnlyTheCLIAndLicense(t *testing.T) {
	distributionDirectory := filepath.Join(t.TempDir(), "snapshot")
	root := repositoryRoot(t)
	result, err := (command.OSRunner{}).Run(context.Background(), command.Request{
		Name: "sh", Arguments: []string{"scripts/release/cli.sh", "snapshot", distributionDirectory}, Directory: root,
	})
	if err != nil {
		t.Fatalf("snapshot command: %v: %s", err, result.Stderr)
	}
	if _, err := os.Stat(distributionDirectory); err != nil {
		t.Fatal(err)
	}
	archivePaths := make(map[string]string, len(expectedArchives))
	for platform, expectedFiles := range expectedArchives {
		extension := ".tar.gz"
		if strings.HasPrefix(platform, "windows_") {
			extension = ".zip"
		}
		archivePath := singleMatch(t, filepath.Join(distributionDirectory, "rm-relay_*_"+platform+extension))
		archivePaths[filepath.Base(archivePath)] = archivePath
		assertArchiveFiles(t, archivePath, expectedFiles)
	}
	assertChecksums(t, distributionDirectory, archivePaths)
}

func singleMatch(t *testing.T, pattern string) string {
	t.Helper()
	matches, err := filepath.Glob(pattern)
	if err != nil {
		t.Fatalf("match snapshot archive %q: %v", pattern, err)
	}
	if len(matches) != 1 {
		t.Fatalf("snapshot archive pattern %q matched %d files", pattern, len(matches))
	}
	return matches[0]
}

func assertArchiveFiles(t *testing.T, archivePath string, expectedFiles []string) {
	t.Helper()
	contents := readArchive(t, archivePath)
	if len(contents) != len(expectedFiles) {
		t.Fatalf("archive %s contains %v, want %v", filepath.Base(archivePath), mapKeys(contents), expectedFiles)
	}
	for _, expectedFile := range expectedFiles {
		contentsSize, exists := contents[expectedFile]
		if !exists {
			t.Errorf("archive %s is missing %s", filepath.Base(archivePath), expectedFile)
			continue
		}
		if contentsSize == 0 {
			t.Errorf("archive %s contains empty %s", filepath.Base(archivePath), expectedFile)
		}
	}
}

func readArchive(t *testing.T, archivePath string) map[string]int64 {
	t.Helper()
	if strings.HasSuffix(archivePath, ".zip") {
		return readZip(t, archivePath)
	}
	return readTarGzip(t, archivePath)
}

func readZip(t *testing.T, archivePath string) map[string]int64 {
	t.Helper()
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		t.Fatalf("open %s: %v", archivePath, err)
	}
	defer reader.Close()
	contents := make(map[string]int64, len(reader.File))
	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			contents[file.Name] = 0
			continue
		}
		contents[file.Name] = int64(file.UncompressedSize64)
	}
	return contents
}

func readTarGzip(t *testing.T, archivePath string) map[string]int64 {
	t.Helper()
	archive, err := os.Open(archivePath)
	if err != nil {
		t.Fatalf("open %s: %v", archivePath, err)
	}
	defer archive.Close()
	gzipReader, err := gzip.NewReader(archive)
	if err != nil {
		t.Fatalf("open gzip stream %s: %v", archivePath, err)
	}
	defer gzipReader.Close()
	contents := make(map[string]int64)
	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("read %s: %v", archivePath, err)
		}
		contents[header.Name] = header.Size
	}
	return contents
}

func assertChecksums(t *testing.T, distributionDirectory string, archivePaths map[string]string) {
	t.Helper()
	checksumPath := singleMatch(t, filepath.Join(distributionDirectory, "rm-relay_*_checksums.txt"))
	checksumFile, err := os.Open(checksumPath)
	if err != nil {
		t.Fatal(err)
	}
	defer checksumFile.Close()
	declaredChecksums := make(map[string]string, len(archivePaths))
	scanner := bufio.NewScanner(checksumFile)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) != 2 {
			t.Fatalf("invalid checksum line %q", scanner.Text())
		}
		declaredChecksums[fields[1]] = fields[0]
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("read checksum file: %v", err)
	}
	if len(declaredChecksums) != len(archivePaths) {
		t.Fatalf("checksum entries = %v, want one for each archive", mapKeys(declaredChecksums))
	}
	for archiveName, archivePath := range archivePaths {
		declared, exists := declaredChecksums[archiveName]
		if !exists {
			t.Errorf("checksum file is missing %s", archiveName)
			continue
		}
		if actual := fileSHA256(t, archivePath); actual != declared {
			t.Errorf("checksum for %s = %s, want %s", archiveName, declared, actual)
		}
	}
}

func fileSHA256(t *testing.T, path string) string {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		t.Fatalf("hash %s: %v", path, err)
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func mapKeys[Value any](values map[string]Value) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	return keys
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, sourcePath, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate distribution test source")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(sourcePath), "..", ".."))
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatal(fmt.Errorf("locate repository root: %w", err))
	}
	return root
}
