package docker

import (
	"strings"
	"testing"
)

func TestGenerateDockerfile(t *testing.T) {
	opts := BuildOptions{
		BaseImage: "ubuntu:latest",
		Features: []FeatureInstallOptions{
			{
				ID:          "ghcr.io/devcontainers/features/git:1",
				UnpackedDir: "/tmp/git-feature",
				Options: map[string]interface{}{
					"version": "latest",
				},
			},
		},
		MetadataLabel: `[{"id": "git"}]`,
	}

	dockerfile, err := GenerateDockerfile(opts)
	if err != nil {
		t.Fatalf("GenerateDockerfile returned unexpected error: %v", err)
	}

	// Verify FROM instruction
	if !strings.Contains(dockerfile, "FROM ubuntu:latest") {
		t.Errorf("Expected Dockerfile to contain 'FROM ubuntu:latest', got:\n%s", dockerfile)
	}

	// Verify COPY feature instruction
	if !strings.Contains(dockerfile, "COPY features/GHCR_IO_DEVCONTAINERS_FEATURES_GIT_1 /tmp/features/GHCR_IO_DEVCONTAINERS_FEATURES_GIT_1") {
		t.Errorf("Expected Dockerfile to contain COPY instruction, got:\n%s", dockerfile)
	}

	// Verify RUN script execution and options environment variables injection
	if !strings.Contains(dockerfile, "VERSION=\"latest\"") || !strings.Contains(dockerfile, "./install.sh") {
		t.Errorf("Expected Dockerfile to inject env and run install.sh, got:\n%s", dockerfile)
	}

	// Verify LABEL injection
	if !strings.Contains(dockerfile, `LABEL devcontainer.metadata="[{\"id\": \"git\"}]"`) {
		t.Errorf("Expected Dockerfile to inject devcontainer.metadata label, got:\n%s", dockerfile)
	}
}

func TestBuildImageCmd(t *testing.T) {
	mock := &mockCmdRunner{}
	cli := NewCLI("podman", "podman-compose", mock.Run)

	opts := BuildOptions{
		BaseImage:   "node:18",
		TargetImage: "my-node-features",
		Features:    []FeatureInstallOptions{},
	}

	// Temporary path for generated Dockerfile
	tmpDockerfile := "/tmp/dc-build-123/Dockerfile"
	tmpContextDir := "/tmp/dc-build-123"

	err := cli.BuildImage(tmpDockerfile, tmpContextDir, opts)
	if err != nil {
		t.Fatalf("BuildImage returned unexpected error: %v", err)
	}

	// Expect podman build -f /tmp/dc-build-123/Dockerfile -t my-node-features /tmp/dc-build-123
	expected := []string{"podman", "build", "-f", tmpDockerfile, "-t", "my-node-features", tmpContextDir}
	if !reflectEqual(mock.lastArgs, expected) {
		t.Errorf("Expected build args %v, got %v", expected, mock.lastArgs)
	}
}

func reflectEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
