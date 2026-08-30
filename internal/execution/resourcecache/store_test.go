package resourcecache

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestMaterializeUsesContentAddressedStablePath(t *testing.T) {
	store := Store{Root: t.TempDir()}

	first, err := store.Materialize("openocd/board", "robomaster-c.cfg", []byte("adapter speed 1800\n"))
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.Materialize("openocd/board", "robomaster-c.cfg", []byte("adapter speed 1800\n"))
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("paths differ for identical content: %q != %q", first, second)
	}
	content, err := os.ReadFile(first)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "adapter speed 1800\n" {
		t.Fatalf("content = %q", content)
	}
}

func TestMaterializeSeparatesContentVersions(t *testing.T) {
	store := Store{Root: t.TempDir()}
	first, err := store.Materialize("mise", "base.toml", []byte("one"))
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.Materialize("mise", "base.toml", []byte("two"))
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("different content reused the same cache path")
	}
}

func TestMaterializeRejectsEscapingLogicalPaths(t *testing.T) {
	store := Store{Root: t.TempDir()}
	_, err := store.Materialize("../outside", "file", []byte("value"))
	if err == nil || !strings.Contains(err.Error(), "namespace") {
		t.Fatalf("Materialize() error = %v", err)
	}
	_, err = store.Materialize("mise", filepath.Join("..", "outside"), []byte("value"))
	if err == nil || !strings.Contains(err.Error(), "name") {
		t.Fatalf("Materialize() error = %v", err)
	}
}

func TestMaterializeIsIdempotentForConcurrentWriters(t *testing.T) {
	store := Store{Root: t.TempDir()}
	const writers = 16
	paths := make(chan string, writers)
	errors := make(chan error, writers)
	var group sync.WaitGroup
	for range writers {
		group.Add(1)
		go func() {
			defer group.Done()
			path, err := store.Materialize("mise", "base.toml", []byte("[tools]\n"))
			paths <- path
			errors <- err
		}()
	}
	group.Wait()
	close(paths)
	close(errors)

	for err := range errors {
		if err != nil {
			t.Fatalf("concurrent Materialize() error = %v", err)
		}
	}
	var expected string
	for path := range paths {
		if expected == "" {
			expected = path
			continue
		}
		if path != expected {
			t.Fatalf("concurrent paths differ: %q != %q", path, expected)
		}
	}
}
