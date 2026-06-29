package cli

import (
	"fmt"

	"github.com/devcontainers/dc/pkg/docker"
)

type UpOptions struct {
	DockerCLI         *docker.CLI
	WorkspaceFolder   string
	ContainerName     string
	BaseImage         string
	OnCreateCommand   interface{}
	PostCreateCommand interface{}
}

func parseCommand(cmd interface{}) ([]string, bool) {
	if cmd == nil {
		return nil, false
	}
	if s, ok := cmd.(string); ok {
		if s == "" {
			return nil, false
		}
		return []string{"sh", "-c", s}, true
	}
	if slice, ok := cmd.([]string); ok {
		if len(slice) == 0 {
			return nil, false
		}
		return slice, true
	}
	if slice, ok := cmd.([]interface{}); ok {
		if len(slice) == 0 {
			return nil, false
		}
		var strSlice []string
		for _, v := range slice {
			strSlice = append(strSlice, fmt.Sprintf("%v", v))
		}
		return strSlice, true
	}
	return nil, false
}

func RunUp(opts UpOptions) error {
	// 1. Inspect container
	_, err := opts.DockerCLI.InspectContainer(opts.ContainerName)
	containerExists := err == nil

	if !containerExists {
		// 2. Inspect base image
		_, err = opts.DockerCLI.InspectImage(opts.BaseImage)
		if err != nil {
			return fmt.Errorf("failed to inspect base image %s: %w", opts.BaseImage, err)
		}

		// 3. Run container
		runOpts := docker.RunOptions{
			Image: opts.BaseImage,
			Name:  opts.ContainerName,
		}
		_, err = opts.DockerCLI.RunContainer(runOpts)
		if err != nil {
			return fmt.Errorf("failed to start container %s: %w", opts.ContainerName, err)
		}
	}

	// 4. Exec OnCreateCommand
	if cmdArgs, ok := parseCommand(opts.OnCreateCommand); ok {
		_, err = opts.DockerCLI.ExecCommand(opts.ContainerName, cmdArgs)
		if err != nil {
			return fmt.Errorf("failed to execute onCreateCommand: %w", err)
		}
	}

	// 5. Exec PostCreateCommand
	if cmdArgs, ok := parseCommand(opts.PostCreateCommand); ok {
		_, err = opts.DockerCLI.ExecCommand(opts.ContainerName, cmdArgs)
		if err != nil {
			return fmt.Errorf("failed to execute postCreateCommand: %w", err)
		}
	}

	return nil
}
