package cli

import "github.com/devcontainers/dc/pkg/docker"

type ExecOptions struct {
	DockerCLI     *docker.CLI
	ContainerName string
	Command       []string
}

func RunExec(opts ExecOptions) error {
	_, err := opts.DockerCLI.ExecCommand(opts.ContainerName, opts.Command)
	return err
}
