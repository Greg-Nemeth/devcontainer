package cli

import (
	"fmt"
	"os"

	"github.com/devcontainers/dc/pkg/docker"
	"github.com/devcontainers/dc/pkg/utils"
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
		// 2. Inspect base image (and pull if not present)
		_, err = opts.DockerCLI.InspectImage(opts.BaseImage)
		if err != nil {
			fmt.Printf("Base image %s not found locally. Pulling...\n", opts.BaseImage)
			if pullErr := opts.DockerCLI.PullImage(opts.BaseImage); pullErr != nil {
				return fmt.Errorf("failed to pull base image %s: %w (inspect error: %v)", opts.BaseImage, pullErr, err)
			}
		}

		// 3. Configure RunOptions with mounts and env
		runOpts := docker.RunOptions{
			Image: opts.BaseImage,
			Name:  opts.ContainerName,
			Env:   make(map[string]string),
		}

		// Expose SSH Agent socket if present
		if sshAuthSock := os.Getenv("SSH_AUTH_SOCK"); sshAuthSock != "" {
			translatedSSH := utils.TranslateWSLPath(sshAuthSock)
			runOpts.Mounts = append(runOpts.Mounts, fmt.Sprintf("type=bind,source=%s,target=/tmp/vscode-ssh-auth.sock", translatedSSH))
			runOpts.Env["SSH_AUTH_SOCK"] = "/tmp/vscode-ssh-auth.sock"
		}

		// Expose GPG Agent socket if present
		if gpgSock := gpgAgentSocketDetector(); gpgSock != "" {
			if _, err := os.Stat(gpgSock); err == nil {
				translatedGPG := utils.TranslateWSLPath(gpgSock)
				runOpts.Mounts = append(runOpts.Mounts, fmt.Sprintf("type=bind,source=%s,target=/tmp/gpg-agent.sock", translatedGPG))
				runOpts.Env["GPG_AGENT_SOCK"] = "/tmp/gpg-agent.sock"
			}
		}

		// Run container with standard keep-alive command
		runOpts.Cmd = []string{"sh", "-c", "echo Container started; trap \"exit 0\" 15; while sleep 1 & wait $!; do :; done"}
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
