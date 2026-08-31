package candidate

import (
	"context"
	"fmt"
	"strings"

	"github.com/x12315/rm-relay/internal/execution/command"
)

func (service Service) currentTaggedImage(ctx context.Context) (string, error) {
	result, err := service.Runner.Run(ctx, command.Request{Name: "docker", Arguments: []string{"image", "ls", "--quiet", "--no-trunc", developmentImageReference}})
	if err != nil {
		return "", candidateProcessFailure("inspect previous development image", result, err)
	}
	fields := strings.Fields(result.Stdout)
	if len(fields) > 1 {
		return "", fmt.Errorf("development image tag resolved to %d identities", len(fields))
	}
	if len(fields) == 0 {
		return "", nil
	}
	return fields[0], nil
}

func (service Service) buildDevelopmentImage(ctx context.Context, repositoryRoot string) (string, error) {
	result, err := service.Runner.Run(ctx, command.Request{
		Name:      "mise",
		Arguments: []string{"run", "environment:embedded:load"},
		Directory: repositoryRoot,
		Stdout:    service.Stdout,
		Stderr:    service.Stderr,
	})
	if err != nil {
		return "", candidateProcessFailure("build candidate development image", result, err)
	}
	return service.inspectImage(ctx)
}

func (service Service) inspectImage(ctx context.Context) (string, error) {
	result, err := service.Runner.Run(ctx, command.Request{Name: "docker", Arguments: []string{"image", "inspect", "--format", "{{.Id}}", developmentImageReference}})
	if err != nil {
		return "", candidateProcessFailure("inspect candidate development image", result, err)
	}
	return oneIdentity("development image", result.Stdout)
}

func (service Service) restoreImage(ctx context.Context, previousImageID string) error {
	arguments := []string{"image", "rm", developmentImageReference}
	action := "remove candidate development image tag"
	if previousImageID != "" {
		arguments = []string{"image", "tag", previousImageID, developmentImageReference}
		action = "restore previous development image tag"
	} else {
		currentImageID, err := service.currentTaggedImage(ctx)
		if err != nil {
			return err
		}
		if currentImageID == "" {
			return nil
		}
	}
	result, err := service.Runner.Run(ctx, command.Request{Name: "docker", Arguments: arguments, Stdout: service.Stdout, Stderr: service.Stderr})
	if err != nil {
		return candidateProcessFailure(action, result, err)
	}
	return nil
}
