package cli

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/devcontainers/dc/pkg/docker"
)

type mockDockerRunner struct {
	commandsRun [][]string
}

func (m *mockDockerRunner) Run(path string, args ...string) ([]byte, error) {
	cmd := append([]string{path}, args...)
	m.commandsRun = append(m.commandsRun, cmd)

	// Mock responses
	if len(args) > 0 && args[0] == "inspect" {
		// Mock container not found (returns error for inspect of non-existent container)
		if args[1] == "non-existent-container" {
			return nil, fmt.Errorf("container not found")
		}
		// Mock image inspect (returns dummy config)
		return []byte(`[{"Id": "sha256:12345", "Config": {"Labels": {}}}]`), nil
	}
	return []byte("mock-output"), nil
}

func TestRunUpWorkflow(t *testing.T) {
	// Clear environment for deterministic test
	os.Setenv("SSH_AUTH_SOCK", "")
	oldDetector := gpgAgentSocketDetector
	gpgAgentSocketDetector = func() string { return "" }
	defer func() { gpgAgentSocketDetector = oldDetector }()

	runner := &mockDockerRunner{}
	dockerCLI := docker.NewCLI("podman", "", runner.Run)

	opts := UpOptions{
		DockerCLI:         dockerCLI,
		WorkspaceFolder:   "/my-workspace",
		ContainerName:     "non-existent-container",
		BaseImage:         "ubuntu:latest",
		OnCreateCommand:   "echo creating",
		PostCreateCommand: []string{"touch", "/done.txt"},
	}

	err := RunUp(opts)
	if err != nil {
		t.Fatalf("RunUp returned unexpected error: %v", err)
	}

	// Verify command sequence:
	// 1. inspect container (non-existent-container)
	// 2. inspect base image (ubuntu:latest)
	// 3. run container (podman run -d --name non-existent-container ubuntu:latest)
	// 4. exec OnCreateCommand
	// 5. exec PostCreateCommand
	expectedCalls := [][]string{
		{"podman", "inspect", "non-existent-container"},
		{"podman", "inspect", "ubuntu:latest"},
		{"podman", "run", "-d", "--name", "non-existent-container", "ubuntu:latest"},
		{"podman", "exec", "non-existent-container", "sh", "-c", "echo creating"},
		{"podman", "exec", "non-existent-container", "touch", "/done.txt"},
	}

	if len(runner.commandsRun) != len(expectedCalls) {
		t.Fatalf("Expected %d docker calls, got %d:\n%v", len(expectedCalls), len(runner.commandsRun), runner.commandsRun)
	}

	for i, gotCall := range runner.commandsRun {
		// Match prefixes or args depending on complexity
		expected := expectedCalls[i]
		if !reflect.DeepEqual(gotCall, expected) {
			t.Errorf("Call %d mismatch.\nGot:  %v\nWant: %v", i, gotCall, expected)
		}
	}
}

func TestRunExec(t *testing.T) {
	runner := &mockDockerRunner{}
	dockerCLI := docker.NewCLI("docker", "", runner.Run)

	var interactiveArgs []string
	dockerCLI.SetInteractiveRunner(func(path string, args ...string) error {
		interactiveArgs = append([]string{path}, args...)
		return nil
	})

	opts := ExecOptions{
		DockerCLI:     dockerCLI,
		ContainerName: "my-running-container",
		Command:       []string{"uname", "-a"},
	}

	err := RunExec(opts)
	if err != nil {
		t.Fatalf("RunExec returned unexpected error: %v", err)
	}

	var expectedCalls []string
	if docker.IsStdinTerminal() {
		expectedCalls = []string{"docker", "exec", "-it", "my-running-container", "uname", "-a"}
	} else {
		expectedCalls = []string{"docker", "exec", "-i", "my-running-container", "uname", "-a"}
	}

	if !reflect.DeepEqual(interactiveArgs, expectedCalls) {
		t.Errorf("Expected interactive args %v, got %v", expectedCalls, interactiveArgs)
	}
}

func TestRunUpWorkflowWithSockets(t *testing.T) {
	// Create a temporary file to act as the GPG socket path so os.Stat succeeds
	tmpFile, err := os.CreateTemp("", "mock-gpg-socket-*")
	if err != nil {
		t.Fatalf("Failed to create mock socket file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	os.Setenv("SSH_AUTH_SOCK", "/tmp/mock-ssh-agent.sock")
	oldDetector := gpgAgentSocketDetector
	gpgAgentSocketDetector = func() string { return tmpFile.Name() }
	defer func() { gpgAgentSocketDetector = oldDetector }()

	runner := &mockDockerRunner{}
	dockerCLI := docker.NewCLI("podman", "", runner.Run)

	opts := UpOptions{
		DockerCLI:       dockerCLI,
		WorkspaceFolder: "/my-workspace",
		ContainerName:   "non-existent-container",
		BaseImage:       "ubuntu:latest",
	}

	err = RunUp(opts)
	if err != nil {
		t.Fatalf("RunUp returned unexpected error: %v", err)
	}

	// Verify podman run args contain mounts and environment variables
	runCallFound := false
	for _, call := range runner.commandsRun {
		if len(call) > 1 && call[1] == "run" {
			runCallFound = true
			// Check SSH mounts
			mountsStr := strings.Join(call, " ")
			if !strings.Contains(mountsStr, "type=bind,source=/tmp/mock-ssh-agent.sock,target=/tmp/vscode-ssh-auth.sock") {
				t.Errorf("Expected podman run command to contain SSH socket mount, got: %v", call)
			}
			if !strings.Contains(mountsStr, "-e SSH_AUTH_SOCK=/tmp/vscode-ssh-auth.sock") {
				t.Errorf("Expected podman run command to contain SSH env var, got: %v", call)
			}

			// Check GPG mounts
			if !strings.Contains(mountsStr, "type=bind,source="+tmpFile.Name()+",target=/tmp/gpg-agent.sock") {
				t.Errorf("Expected podman run command to contain GPG socket mount, got: %v", call)
			}
			if !strings.Contains(mountsStr, "-e GPG_AGENT_SOCK=/tmp/gpg-agent.sock") {
				t.Errorf("Expected podman run command to contain GPG env var, got: %v", call)
			}
		}
	}

	if !runCallFound {
		t.Fatal("Expected podman run command call, none found")
	}
}
