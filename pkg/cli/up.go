package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/devcontainers/dc/pkg/config"
	"github.com/devcontainers/dc/pkg/docker"
	"github.com/devcontainers/dc/pkg/utils"
)

type UpOptions struct {
	DockerCLI                *docker.CLI
	WorkspaceFolder          string
	ContainerWorkspaceFolder string
	ContainerName            string
	BaseImage                string
	OnCreateCommand          interface{}
	PostCreateCommand        interface{}
	DockerComposeFile        interface{}
	Service                  string
	ConfigPath               string
	Mounts                   []string
	WorkspaceMount           string
	RemoteUser               string
}

// CleanProjectName cleans a string to be compatible with Docker Compose project names
func CleanProjectName(name string) string {
	var sb strings.Builder
	for _, r := range strings.ToLower(name) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			sb.WriteRune(r)
		}
	}
	res := sb.String()
	if res == "" {
		res = "devcontainer"
	}
	return res
}

// ParseStringOrSlice converts a string or []interface{} or []string to a []string
func ParseStringOrSlice(val interface{}) []string {
	if val == nil {
		return nil
	}
	if s, ok := val.(string); ok {
		return []string{s}
	}
	if slice, ok := val.([]interface{}); ok {
		var res []string
		for _, v := range slice {
			if s, ok := v.(string); ok {
				res = append(res, s)
			}
		}
		return res
	}
	if slice, ok := val.([]string); ok {
		return slice
	}
	return nil
}

// ResolveComposeFiles resolves dockerComposeFile paths relative to devcontainer.json directory
func ResolveComposeFiles(dockerComposeFile interface{}, configPath string) []string {
	configDir := filepath.Dir(configPath)
	var resolvedComposeFiles []string
	for _, f := range ParseStringOrSlice(dockerComposeFile) {
		if filepath.IsAbs(f) {
			resolvedComposeFiles = append(resolvedComposeFiles, f)
		} else {
			resolvedComposeFiles = append(resolvedComposeFiles, filepath.Join(configDir, f))
		}
	}
	return resolvedComposeFiles
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
	var targetContainer string

	if opts.DockerComposeFile != nil && opts.Service != "" {
		// Resolve compose files
		resolvedComposeFiles := ResolveComposeFiles(opts.DockerComposeFile, opts.ConfigPath)
		projectName := CleanProjectName(filepath.Base(opts.WorkspaceFolder))

		// Start compose stack
		fmt.Printf("Starting compose stack with project %s...\n", projectName)
		err := opts.DockerCLI.ComposeUp(resolvedComposeFiles, projectName)
		if err != nil {
			return fmt.Errorf("failed to start compose stack: %w", err)
		}

		// Get target service container
		cID, err := opts.DockerCLI.GetComposeServiceContainer(resolvedComposeFiles, projectName, opts.Service)
		if err != nil {
			return fmt.Errorf("failed to get container for service %s: %w", opts.Service, err)
		}
		if cID == "" {
			return fmt.Errorf("service %s has no running container", opts.Service)
		}
		targetContainer = cID
	} else {
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

			// Add workspace mount
			if opts.WorkspaceMount != "" {
				runOpts.Mounts = append(runOpts.Mounts, opts.WorkspaceMount)
			}

			// Add other custom mounts
			runOpts.Mounts = append(runOpts.Mounts, opts.Mounts...)

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
		targetContainer = opts.ContainerName
	}

	// 4. Exec OnCreateCommand
	if cmdArgs, ok := parseCommand(opts.OnCreateCommand); ok {
		_, err := opts.DockerCLI.ExecCommandWithUser(targetContainer, "", opts.ContainerWorkspaceFolder, cmdArgs)
		if err != nil {
			return fmt.Errorf("failed to execute onCreateCommand: %w", err)
		}
	}

	// 5. Exec PostCreateCommand
	if cmdArgs, ok := parseCommand(opts.PostCreateCommand); ok {
		_, err := opts.DockerCLI.ExecCommandWithUser(targetContainer, opts.RemoteUser, opts.ContainerWorkspaceFolder, cmdArgs)
		if err != nil {
			return fmt.Errorf("failed to execute postCreateCommand: %w", err)
		}
	}

	return nil
}

// ResolveContainerName finds the actual container ID if Compose is used, otherwise returns defaultName
func ResolveContainerName(wsFolder string, cfgPath string, defaultName string, dCli *docker.CLI) (string, error) {
	if cfgPath == "" {
		p1 := filepath.Join(wsFolder, ".devcontainer", "devcontainer.json")
		p2 := filepath.Join(wsFolder, ".devcontainer.json")
		if _, err := os.Stat(p1); err == nil {
			cfgPath = p1
		} else if _, err := os.Stat(p2); err == nil {
			cfgPath = p2
		}
	}

	if cfgPath != "" {
		data, err := os.ReadFile(cfgPath)
		if err == nil {
			parsed, err := config.Parse(string(data))
			if err == nil && parsed.DockerComposeFile != nil && parsed.Service != "" {
				resolvedComposeFiles := ResolveComposeFiles(parsed.DockerComposeFile, cfgPath)
				projectName := CleanProjectName(filepath.Base(wsFolder))
				cID, err := dCli.GetComposeServiceContainer(resolvedComposeFiles, projectName, parsed.Service)
				if err == nil && cID != "" {
					return cID, nil
				}
			}
		}
	}
	return defaultName, nil
}
