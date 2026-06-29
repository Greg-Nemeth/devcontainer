package docker

import (
	"reflect"
	"testing"
)

type mockCmdRunner struct {
	lastArgs []string
}

func (m *mockCmdRunner) Run(path string, args ...string) ([]byte, error) {
	m.lastArgs = append([]string{path}, args...)
	return []byte(`[{"Id": "test-container-id", "State": {"Running": true}}]`), nil
}

func TestInspectContainer(t *testing.T) {
	mock := &mockCmdRunner{}
	cli := NewCLI("docker", "docker-compose", mock.Run)

	_, err := cli.InspectContainer("my-container")
	if err != nil {
		t.Fatalf("Unexpected error inspect: %v", err)
	}

	expectedArgs := []string{"docker", "inspect", "my-container"}
	if !reflect.DeepEqual(mock.lastArgs, expectedArgs) {
		t.Errorf("Expected args %v, got %v", expectedArgs, mock.lastArgs)
	}
}

func TestRunContainer(t *testing.T) {
	mock := &mockCmdRunner{}
	cli := NewCLI("podman", "podman-compose", mock.Run)

	options := RunOptions{
		Image: "node:18",
		Name:  "my-node-container",
		Mounts: []string{
			"type=bind,source=/src,target=/workspace",
		},
		Env: map[string]string{
			"NODE_ENV": "development",
		},
	}

	_, err := cli.RunContainer(options)
	if err != nil {
		t.Fatalf("Unexpected error run: %v", err)
	}

	// Order of env variables and arguments should be checked
	// Note: mount and env parameters will be parsed into --mount and -e
	hasMount := false
	hasEnv := false
	hasImage := false
	hasName := false

	for i, arg := range mock.lastArgs {
		if arg == "--mount" && mock.lastArgs[i+1] == "type=bind,source=/src,target=/workspace" {
			hasMount = true
		}
		if arg == "-e" && mock.lastArgs[i+1] == "NODE_ENV=development" {
			hasEnv = true
		}
		if arg == "node:18" {
			hasImage = true
		}
		if arg == "--name" && mock.lastArgs[i+1] == "my-node-container" {
			hasName = true
		}
	}

	if !hasMount || !hasEnv || !hasImage || !hasName {
		t.Errorf("Run container generated arguments are incomplete: %v", mock.lastArgs)
	}
}

func TestComposeUp(t *testing.T) {
	mock := &mockCmdRunner{}
	cli := NewCLI("docker", "docker-compose", mock.Run)

	err := cli.ComposeUp([]string{"docker-compose.yml", "docker-compose.override.yml"}, "my-project")
	if err != nil {
		t.Fatalf("Unexpected compose error: %v", err)
	}

	// We expect docker-compose -f file1 -f file2 -p projectName up -d
	expected := []string{"docker-compose", "-f", "docker-compose.yml", "-f", "docker-compose.override.yml", "-p", "my-project", "up", "-d"}
	if !reflect.DeepEqual(mock.lastArgs, expected) {
		t.Errorf("Expected compose args %v, got %v", expected, mock.lastArgs)
	}
}
