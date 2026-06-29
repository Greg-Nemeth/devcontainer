package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devcontainers/dc/pkg/docker"
)

func detectContainerEngine() (string, bool) {
	if _, err := exec.LookPath("podman-remote"); err == nil {
		return "podman-remote", true
	}
	if _, err := exec.LookPath("podman"); err == nil {
		return "podman", true
	}
	if _, err := exec.LookPath("docker"); err == nil {
		return "docker", true
	}
	return "", false
}

func detectComposeEngine() (string, bool) {
	if _, err := exec.LookPath("podman-compose"); err == nil {
		return "podman-compose", true
	}
	if _, err := exec.LookPath("docker-compose"); err == nil {
		return "docker-compose", true
	}
	return "", false
}

func TestIntegrationSingleContainer(t *testing.T) {
	cliName, ok := detectContainerEngine()
	if !ok {
		t.Skip("No container engine (podman/docker) available, skipping integration test")
	}

	// Verify socket connection is working
	checkCmd := exec.Command(cliName, "info")
	if err := checkCmd.Run(); err != nil {
		t.Skipf("Container engine %s is installed but unreachable/not running, skipping integration test: %v", cliName, err)
	}

	// Create a temp workspace
	wsDir := t.TempDir()

	containerName := "dc-integration-test-container"
	// Ensure any stale container from previous failed run is cleaned up
	exec.Command(cliName, "rm", "-f", containerName).Run()

	dCli := docker.NewCLI(cliName, "", nil)

	opts := UpOptions{
		DockerCLI:         dCli,
		WorkspaceFolder:   wsDir,
		ContainerName:     containerName,
		BaseImage:         "mcr.microsoft.com/devcontainers/base:ubuntu",
		OnCreateCommand:   "echo 'on-create-run' > /tmp/on-create.txt",
		PostCreateCommand: "echo 'post-create-run' > /tmp/post-create.txt",
	}

	// Spin up
	t.Log("Spinning up container...")
	err := RunUp(opts)
	if err != nil {
		t.Fatalf("RunUp integration failed: %v", err)
	}

	// Defer cleanup
	defer func() {
		t.Log("Tearing down container...")
		exec.Command(cliName, "rm", "-f", containerName).Run()
	}()

	// Verify env variables & command hook runs via ExecCommand
	t.Log("Verifying commands inside container...")
	out, err := dCli.ExecCommand(containerName, []string{"cat", "/tmp/on-create.txt"})
	if err != nil {
		t.Fatalf("Failed to verify on-create script inside container: %v", err)
	}
	if strings.TrimSpace(string(out)) != "on-create-run" {
		t.Errorf("Expected '/tmp/on-create.txt' to contain 'on-create-run', got %q", string(out))
	}

	out, err = dCli.ExecCommand(containerName, []string{"cat", "/tmp/post-create.txt"})
	if err != nil {
		t.Fatalf("Failed to verify post-create script inside container: %v", err)
	}
	if strings.TrimSpace(string(out)) != "post-create-run" {
		t.Errorf("Expected '/tmp/post-create.txt' to contain 'post-create-run', got %q", string(out))
	}
}

func TestIntegrationDockerCompose(t *testing.T) {
	cliName, ok := detectContainerEngine()
	if !ok {
		t.Skip("No container engine available, skipping compose integration test")
	}

	composeName, ok := detectComposeEngine()
	if !ok {
		t.Skip("No compose engine (podman-compose/docker-compose) available, skipping compose integration test")
	}

	// Verify docker compose config checks
	checkCmd := exec.Command(composeName, "version")
	if err := checkCmd.Run(); err != nil {
		t.Skipf("Compose engine %s is installed but unreachable, skipping compose integration test: %v", composeName, err)
	}

	wsDir := t.TempDir()

	// Write a simple docker-compose.yml
	composeYAML := `
version: '3'
services:
  db:
    image: alpine:latest
    command: sleep 3600
`
	composePath := filepath.Join(wsDir, "docker-compose.yml")
	err := os.WriteFile(composePath, []byte(composeYAML), 0644)
	if err != nil {
		t.Fatalf("Failed to write docker-compose.yml: %v", err)
	}

	dCli := docker.NewCLI(cliName, composeName, nil)

	projectName := "dc-integration-compose-proj"
	// Ensure clean slate
	dCli.ComposeDown([]string{composePath}, projectName)

	t.Log("Spinning up compose stack...")
	err = dCli.ComposeUp([]string{composePath}, projectName)
	if err != nil {
		t.Fatalf("ComposeUp failed: %v", err)
	}

	// Defer cleanup
	defer func() {
		t.Log("Tearing down compose stack...")
		dCli.ComposeDown([]string{composePath}, projectName)
	}()

	// Query container ID
	t.Log("Verifying service container is running...")
	cID, err := dCli.GetComposeServiceContainer([]string{composePath}, projectName, "db")
	if err != nil {
		t.Fatalf("GetComposeServiceContainer failed: %v", err)
	}

	if cID == "" {
		t.Fatal("Expected service 'db' to return running container ID, got empty string")
	}
}
