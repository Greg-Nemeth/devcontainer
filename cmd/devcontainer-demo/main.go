package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/devcontainers/dc/pkg/config"
	"github.com/devcontainers/dc/pkg/docker"
	"github.com/devcontainers/dc/pkg/logging"
	"github.com/devcontainers/dc/pkg/varsub"
)

func main() {
	workspacePtr := flag.String("workspace-folder", ".", "Path to workspace folder")
	dockerPathPtr := flag.String("docker-path", "podman-remote", "Docker/Podman executable path")
	flag.Parse()

	workspace, err := filepath.Abs(*workspacePtr)
	if err != nil {
		fmt.Printf("Error resolving absolute path: %v\n", err)
		os.Exit(1)
	}

	logger := logging.NewTerminalLogger(os.Stdout, logging.LevelInfo)
	logger.Write(fmt.Sprintf("Loading configuration in: %s", workspace), logging.LevelInfo)

	// Search for config file
	var configPath string
	pathsToTry := []string{
		filepath.Join(workspace, ".devcontainer", "devcontainer.json"),
		filepath.Join(workspace, ".devcontainer.json"),
	}

	for _, p := range pathsToTry {
		if _, err := os.Stat(p); err == nil {
			configPath = p
			break
		}
	}

	if configPath == "" {
		logger.Write("No devcontainer.json found. Creating a temporary dummy .devcontainer/devcontainer.json for demonstration...", logging.LevelInfo)
		err := os.MkdirAll(filepath.Join(workspace, ".devcontainer"), 0755)
		if err != nil {
			fmt.Printf("Failed to create .devcontainer folder: %v\n", err)
			os.Exit(1)
		}
		dummyContent := `{
			// Sample Devcontainer config
			"name": "Demo Go Devcontainer",
			"image": "mcr.microsoft.com/devcontainers/base:ubuntu",
			"forwardPorts": [8080, 9000],
			"containerEnv": {
				"DEVELOPER": "${localEnv:USER:developer_user}",
				"WORKSPACE_BASE": "${localWorkspaceFolderBasename}"
			}
		}`
		configPath = filepath.Join(workspace, ".devcontainer", "devcontainer.json")
		err = os.WriteFile(configPath, []byte(dummyContent), 0644)
		if err != nil {
			fmt.Printf("Failed to write sample devcontainer.json: %v\n", err)
			os.Exit(1)
		}
	}

	// Read and parse config (JSONC comment stripping demonstration)
	data, err := os.ReadFile(configPath)
	if err != nil {
		fmt.Printf("Error reading config file: %v\n", err)
		os.Exit(1)
	}

	parsed, err := config.Parse(string(data))
	if err != nil {
		fmt.Printf("Error parsing configuration: %v\n", err)
		os.Exit(1)
	}

	logger.Write(fmt.Sprintf("Successfully parsed configuration. Name: %q, Base Image: %q", parsed.Name, parsed.Image), logging.LevelInfo)

	// Perform Variable Substitution
	envMap := make(map[string]string)
	for _, env := range os.Environ() {
		parts := filepath.SplitList(env) // split env key value
		if len(parts) > 0 {
			kv := strings.SplitN(env, "=", 2)
			if len(kv) == 2 {
				envMap[kv[0]] = kv[1]
			}
		}
	}

	ctx := varsub.SubstitutionContext{
		Platform:                 "linux",
		ConfigFile:               configPath,
		LocalWorkspaceFolder:     workspace,
		ContainerWorkspaceFolder: "/workspaces/devcontainer",
		Env:                      envMap,
	}

	// Substitute containerEnv map
	substitutedEnv := make(map[string]string)
	for k, v := range parsed.ContainerEnv {
		res := varsub.Substitute(ctx, v)
		if s, ok := res.(string); ok {
			substitutedEnv[k] = s
		}
	}

	logger.Write("Substituted Container Environment variables:", logging.LevelInfo)
	for k, v := range substitutedEnv {
		logger.Write(fmt.Sprintf("  %s = %s", k, v), logging.LevelInfo)
	}

	// Interacting with Podman CLI wrapper
	logger.Write(fmt.Sprintf("Querying Podman CLI at path: %s", *dockerPathPtr), logging.LevelInfo)
	cli := docker.NewCLI(*dockerPathPtr, "", nil)

	start := logger.Start("Inspecting Podman host configuration...", logging.LevelInfo)
	inspectBytes, err := cli.InspectImage(parsed.Image)
	logger.Stop("Finished host inspection.", start, logging.LevelInfo)

	if err != nil {
		logger.Write(fmt.Sprintf("Image %q not found locally. (Error: %v)", parsed.Image, err), logging.LevelWarning)
		logger.Write(fmt.Sprintf("Tip: Run 'CONTAINER_HOST=%s %s pull %s' to download the image.", os.Getenv("CONTAINER_HOST"), *dockerPathPtr, parsed.Image), logging.LevelInfo)
	} else {
		logger.Write(fmt.Sprintf("Successfully inspected image %q. Length of inspect output: %d bytes.", parsed.Image, len(inspectBytes)), logging.LevelInfo)
	}
}
