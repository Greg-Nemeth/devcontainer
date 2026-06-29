package docker

import (
	"reflect"
	"testing"
)

func TestComposeDownCmd(t *testing.T) {
	mock := &mockCmdRunner{}
	cli := NewCLI("docker", "docker-compose", mock.Run)

	err := cli.ComposeDown([]string{"docker-compose.yml"}, "test-project")
	if err != nil {
		t.Fatalf("ComposeDown returned unexpected error: %v", err)
	}

	expected := []string{"docker-compose", "-f", "docker-compose.yml", "-p", "test-project", "down"}
	if !reflect.DeepEqual(mock.lastArgs, expected) {
		t.Errorf("Expected compose down args %v, got %v", expected, mock.lastArgs)
	}
}

func TestGetComposeServiceContainerCmd(t *testing.T) {
	mock := &mockCmdRunner{}
	// Setup mock to return a dummy container ID
	mock.lastArgs = nil
	mockRunner := func(path string, args ...string) ([]byte, error) {
		mock.lastArgs = append([]string{path}, args...)
		return []byte("container-abc-123\n"), nil
	}

	cli := NewCLI("docker", "docker-compose", mockRunner)

	containerID, err := cli.GetComposeServiceContainer([]string{"docker-compose.yml"}, "test-project", "app")
	if err != nil {
		t.Fatalf("GetComposeServiceContainer returned unexpected error: %v", err)
	}

	if containerID != "container-abc-123" {
		t.Errorf("Expected container ID 'container-abc-123', got %q", containerID)
	}

	expected := []string{"docker-compose", "-f", "docker-compose.yml", "-p", "test-project", "ps", "-q", "app"}
	if !reflect.DeepEqual(mock.lastArgs, expected) {
		t.Errorf("Expected compose ps args %v, got %v", expected, mock.lastArgs)
	}
}
