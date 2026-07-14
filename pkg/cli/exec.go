package cli

import (
	"github.com/devcontainers/dc/pkg/docker"
)

type ExecOptions struct {
	DockerCLI                *docker.CLI
	ContainerName            string
	User                     string
	ContainerWorkspaceFolder string
	Command                  []string
}

func RunExec(opts ExecOptions) error {
	return opts.DockerCLI.ExecInteractive(opts.ContainerName, opts.User, opts.ContainerWorkspaceFolder, opts.Command)
}
