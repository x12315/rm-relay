package architecture

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"
)

const internalImportPrefix = "github.com/x12315/rm-relay/internal/"

var allowedDependencies = map[string][]string{
	"build":     {"build", "builder", "execution", "profile", "project"},
	"builder":   {"builder", "execution"},
	"cli":       {"build", "builder", "cli", "execution", "profile", "project", "target"},
	"execution": {"execution"},
	"profile":   {},
	"project":   {},
	"target":    {"build", "execution", "profile", "target"},
}

func TestInternalModulesFollowDependencyDirection(t *testing.T) {
	repositoryRoot := locateRepositoryRoot(t)
	internalRoot := filepath.Join(repositoryRoot, "internal")
	err := filepath.WalkDir(internalRoot, func(sourcePath string, entry fs.DirEntry, walkError error) error {
		if walkError != nil {
			return walkError
		}
		if entry.IsDir() || filepath.Ext(sourcePath) != ".go" || strings.HasSuffix(sourcePath, "_test.go") {
			return nil
		}
		relativePath, err := filepath.Rel(internalRoot, sourcePath)
		if err != nil {
			return err
		}
		owner := strings.Split(filepath.ToSlash(relativePath), "/")[0]
		allowed, knownOwner := allowedDependencies[owner]
		if !knownOwner {
			t.Errorf("production package %q has no declared module owner", relativePath)
			return nil
		}

		parsed, err := parser.ParseFile(token.NewFileSet(), sourcePath, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imported := range parsed.Imports {
			importPath, unquoteError := strconv.Unquote(imported.Path.Value)
			if unquoteError != nil {
				return unquoteError
			}
			if strings.HasPrefix(importPath, "github.com/x12315/rm-relay/tests/support") {
				t.Errorf("production package %q imports test support %s", relativePath, importPath)
			}
			assertAllowedInternalImport(t, relativePath, owner, allowed, imported)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("inspect internal module dependencies: %v", err)
	}
}

func TestDockerAndBuildxAdaptersDependOnlyOnCommandBoundary(t *testing.T) {
	root := filepath.Join(locateRepositoryRoot(t), "internal", "execution")
	for _, adapter := range []string{"docker", "buildx"} {
		err := filepath.WalkDir(filepath.Join(root, adapter), func(sourcePath string, entry fs.DirEntry, walkError error) error {
			if walkError != nil {
				return walkError
			}
			if entry.IsDir() || filepath.Ext(sourcePath) != ".go" || strings.HasSuffix(sourcePath, "_test.go") {
				return nil
			}
			parsed, err := parser.ParseFile(token.NewFileSet(), sourcePath, nil, parser.ImportsOnly)
			if err != nil {
				return err
			}
			for _, imported := range parsed.Imports {
				path, err := strconv.Unquote(imported.Path.Value)
				if err != nil {
					return err
				}
				if strings.HasPrefix(path, internalImportPrefix) && path != internalImportPrefix+"execution/command" {
					t.Errorf("%s adapter imports non-command internal package %s", adapter, path)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

func assertAllowedInternalImport(t *testing.T, sourcePath, owner string, allowed []string, imported *ast.ImportSpec) {
	t.Helper()
	importPath, err := strconv.Unquote(imported.Path.Value)
	if err != nil {
		t.Errorf("decode import in %s: %v", sourcePath, err)
		return
	}
	if !strings.HasPrefix(importPath, internalImportPrefix) {
		return
	}
	dependency := strings.Split(strings.TrimPrefix(importPath, internalImportPrefix), "/")[0]
	if !slices.Contains(allowed, dependency) {
		t.Errorf("module %q must not depend on %q: %s imports %s", owner, dependency, sourcePath, importPath)
	}
}

func locateRepositoryRoot(t *testing.T) string {
	t.Helper()
	_, sourcePath, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate dependency contract source")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(sourcePath), "..", ".."))
}
