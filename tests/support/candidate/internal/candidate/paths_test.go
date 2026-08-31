package candidate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveLayoutUsesCanonicalRepositoryIdentity(t *testing.T) {
	repositoryRoot := filepath.Join(t.TempDir(), "repository")
	if err := os.Mkdir(repositoryRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	repositoryAlias := filepath.Join(t.TempDir(), "repository-alias")
	if err := os.Symlink(repositoryRoot, repositoryAlias); err != nil {
		t.Skipf("create repository symlink: %v", err)
	}
	cacheRoot := t.TempDir()

	direct, err := ResolveLayout(repositoryRoot, cacheRoot)
	if err != nil {
		t.Fatal(err)
	}
	aliased, err := ResolveLayout(repositoryAlias, cacheRoot)
	if err != nil {
		t.Fatal(err)
	}

	if direct.RepositoryRoot != aliased.RepositoryRoot || direct.RepositoryKey != aliased.RepositoryKey || direct.Root != aliased.Root {
		t.Fatalf("canonical layouts differ: direct=%#v alias=%#v", direct, aliased)
	}
	resolvedCacheRoot, err := filepath.EvalSymlinks(cacheRoot)
	if err != nil {
		t.Fatal(err)
	}
	wantParent := filepath.Join(resolvedCacheRoot, "rm-relay", "experience")
	if filepath.Dir(direct.Root) != wantParent {
		t.Fatalf("layout parent = %q, want %q", filepath.Dir(direct.Root), wantParent)
	}
	if direct.ConfigDirectory != filepath.Join(direct.Root, "config") {
		t.Fatalf("config directory = %q", direct.ConfigDirectory)
	}
}

func TestResolveLayoutSeparatesRepositories(t *testing.T) {
	cacheRoot := t.TempDir()
	first, err := ResolveLayout(t.TempDir(), cacheRoot)
	if err != nil {
		t.Fatal(err)
	}
	second, err := ResolveLayout(t.TempDir(), cacheRoot)
	if err != nil {
		t.Fatal(err)
	}
	if first.RepositoryKey == second.RepositoryKey || first.Root == second.Root {
		t.Fatalf("different repositories share layout: first=%#v second=%#v", first, second)
	}
}

func TestResolveLayoutRejectsCacheInsideRepository(t *testing.T) {
	repositoryRoot := t.TempDir()

	_, err := ResolveLayout(repositoryRoot, filepath.Join(repositoryRoot, ".cache"))

	if err == nil {
		t.Fatal("ResolveLayout() accepted a cache path inside the repository")
	}
}

func TestLayoutWithRootMovesEveryCandidateOwnedPath(t *testing.T) {
	original := Layout{BinaryPath: filepath.Join("original", "bin", "rm-relay")}
	root := filepath.Join("cache", "candidate")

	moved := layoutWithRoot(original, root)

	for name, path := range map[string]string{
		"state":     moved.StatePath,
		"binary":    moved.BinaryPath,
		"config":    moved.ConfigDirectory,
		"template":  moved.TemplateOrigin,
		"workspace": moved.Workspace,
		"logs":      moved.Logs,
	} {
		if !pathWithin(root, path) {
			t.Fatalf("%s path %q is outside moved root %q", name, path, root)
		}
	}
}
