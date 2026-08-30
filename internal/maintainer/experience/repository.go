package experience

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/x12315/rm-relay/internal/execution/command"
)

const templatePath = "project-templates/cross-platform-cpp"

func (service Service) requireCleanRepository(ctx context.Context, repositoryRoot string) error {
	result, err := service.Runner.Run(ctx, command.Request{Name: "git", Arguments: []string{"status", "--porcelain"}, Directory: repositoryRoot})
	if err != nil {
		return candidateProcessFailure("inspect repository status", result, err)
	}
	if strings.TrimSpace(result.Stdout) != "" {
		return fmt.Errorf("repository contains uncommitted changes; commit or remove them before preparing or entering a candidate")
	}
	return nil
}

func (service Service) gitRevision(ctx context.Context, repositoryRoot string) (string, error) {
	result, err := service.Runner.Run(ctx, command.Request{Name: "git", Arguments: []string{"rev-parse", "HEAD"}, Directory: repositoryRoot})
	if err != nil {
		return "", candidateProcessFailure("read repository revision", result, err)
	}
	return oneIdentity("repository revision", result.Stdout)
}

func (service Service) createTemplateOrigin(ctx context.Context, layout Layout, candidateRevision string) (string, error) {
	filesResult, err := service.Runner.Run(ctx, command.Request{
		Name:      "git",
		Arguments: []string{"ls-files", "-z", "--", templatePath},
		Directory: layout.RepositoryRoot,
	})
	if err != nil {
		return "", candidateProcessFailure("list committed Project Template files", filesResult, err)
	}
	worktree := filepath.Join(layout.Root, ".template-worktree")
	if err := os.MkdirAll(worktree, 0o755); err != nil {
		return "", fmt.Errorf("create Project Template worktree: %w", err)
	}
	defer os.RemoveAll(worktree)
	fileCount := 0
	for _, repositoryPath := range strings.Split(filesResult.Stdout, "\x00") {
		if repositoryPath == "" {
			continue
		}
		relativePath, found := strings.CutPrefix(filepath.ToSlash(repositoryPath), templatePath+"/")
		if !found || !filepath.IsLocal(filepath.FromSlash(relativePath)) {
			return "", fmt.Errorf("Git returned unsafe Project Template path %q", repositoryPath)
		}
		sourcePath := filepath.Join(layout.RepositoryRoot, filepath.FromSlash(repositoryPath))
		destinationPath := filepath.Join(worktree, filepath.FromSlash(relativePath))
		if err := copyTemplateFile(sourcePath, destinationPath); err != nil {
			return "", err
		}
		fileCount++
	}
	if fileCount == 0 {
		return "", fmt.Errorf("Project Template contains no committed files")
	}
	commands := [][]string{
		{"init", "--quiet"},
		{"config", "user.name", "RM Relay Candidate"},
		{"config", "user.email", "candidate@rm-relay.invalid"},
		{"add", "."},
		{"commit", "--quiet", "-m", "RM Relay candidate " + candidateRevision},
	}
	for _, arguments := range commands {
		result, err := service.Runner.Run(ctx, command.Request{Name: "git", Arguments: arguments, Directory: worktree})
		if err != nil {
			return "", candidateProcessFailure("create candidate Project Template repository", result, err)
		}
	}
	templateRevision, err := service.gitRevision(ctx, worktree)
	if err != nil {
		return "", err
	}
	result, err := service.Runner.Run(ctx, command.Request{Name: "git", Arguments: []string{"clone", "--quiet", "--bare", worktree, layout.TemplateOrigin}})
	if err != nil {
		return "", candidateProcessFailure("create candidate Project Template origin", result, err)
	}
	return templateRevision, nil
}

func (service Service) bareRepositoryRevision(ctx context.Context, repositoryPath string) (string, error) {
	result, err := service.Runner.Run(ctx, command.Request{Name: "git", Arguments: []string{"--git-dir", repositoryPath, "rev-parse", "HEAD"}})
	if err != nil {
		return "", candidateProcessFailure("read candidate Project Template revision", result, err)
	}
	return oneIdentity("Project Template revision", result.Stdout)
}

func copyTemplateFile(sourcePath, destinationPath string) error {
	info, err := os.Lstat(sourcePath)
	if err != nil {
		return fmt.Errorf("inspect Project Template file: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return fmt.Errorf("Project Template path must be a regular file: %s", sourcePath)
	}
	if err := os.MkdirAll(filepath.Dir(destinationPath), 0o755); err != nil {
		return fmt.Errorf("create Project Template directory: %w", err)
	}
	source, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("open Project Template file: %w", err)
	}
	defer source.Close()
	destination, err := os.OpenFile(destinationPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, info.Mode().Perm())
	if err != nil {
		return fmt.Errorf("create Project Template file: %w", err)
	}
	if _, err := io.Copy(destination, source); err != nil {
		destination.Close()
		return fmt.Errorf("copy Project Template file: %w", err)
	}
	if err := destination.Close(); err != nil {
		return fmt.Errorf("close Project Template file: %w", err)
	}
	return nil
}
