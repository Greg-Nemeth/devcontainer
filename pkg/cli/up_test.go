package cli

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/devcontainers/dc/pkg/docker"
	"github.com/devcontainers/dc/pkg/features"
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

	for _, arg := range args {
		if arg == "config" {
			return []byte("services:\n  db:\n    image: alpine:latest\n"), nil
		}
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
		{"podman", "run", "-d", "--name", "non-existent-container", "ubuntu:latest", "sh", "-c", "echo Container started; trap \"exit 0\" 15; while sleep 1 & wait $!; do :; done"},
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

func TestRunUpWorkflowWithFeatures(t *testing.T) {
	// Create dummy tarball bytes representing feature payload
	var tarballBuf bytes.Buffer
	gw := gzip.NewWriter(&tarballBuf)
	tw := tar.NewWriter(gw)

	// Add install.sh
	installShContent := "echo 'installing feature'"
	hdr1 := &tar.Header{
		Name: "install.sh",
		Mode: 0755,
		Size: int64(len(installShContent)),
	}
	if err := tw.WriteHeader(hdr1); err != nil {
		t.Fatalf("Failed to write install.sh header: %v", err)
	}
	if _, err := tw.Write([]byte(installShContent)); err != nil {
		t.Fatalf("Failed to write install.sh content: %v", err)
	}

	// Add devcontainer-feature.json
	metaContent := `{"id": "git", "installsAfter": []}`
	hdr2 := &tar.Header{
		Name: "devcontainer-feature.json",
		Mode: 0644,
		Size: int64(len(metaContent)),
	}
	if err := tw.WriteHeader(hdr2); err != nil {
		t.Fatalf("Failed to write metadata header: %v", err)
	}
	if _, err := tw.Write([]byte(metaContent)); err != nil {
		t.Fatalf("Failed to write metadata content: %v", err)
	}

	tw.Close()
	gw.Close()

	tarballBytes := tarballBuf.Bytes()

	// Start local mock registry server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Mock token auth challenge
		if r.URL.Path != "/token" && r.Header.Get("Authorization") == "" {
			w.Header().Set("Www-Authenticate", `Bearer realm="http://`+r.Host+`/token",service="mock-registry",scope="repository:devcontainers/features/git:pull"`)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if r.URL.Path == "/token" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"token": "mock-access-token"}`))
			return
		}

		if strings.Contains(r.URL.Path, "/manifests/") {
			w.Header().Set("Content-Type", features.OciManifestMediaType)
			w.Write([]byte(`{
				"schemaVersion": 2,
				"mediaType": "application/vnd.oci.image.manifest.v1+json",
				"layers": [
					{
						"mediaType": "application/vnd.devcontainers.layer.v1+tar+gzip",
						"digest": "sha256:abc123digest",
						"size": 1234
					}
				]
			}`))
			return
		}

		if strings.Contains(r.URL.Path, "/blobs/") {
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write(tarballBytes)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	urlHost := strings.TrimPrefix(server.URL, "http://")

	os.Setenv("SSH_AUTH_SOCK", "")
	oldDetector := gpgAgentSocketDetector
	gpgAgentSocketDetector = func() string { return "" }
	defer func() { gpgAgentSocketDetector = oldDetector }()

	runner := &mockDockerRunner{}
	dockerCLI := docker.NewCLI("podman", "", runner.Run)

	featureRef := urlHost + "/devcontainers/features/git:1"
	opts := UpOptions{
		DockerCLI:       dockerCLI,
		WorkspaceFolder: "/my-workspace",
		ContainerName:   "non-existent-container",
		BaseImage:       "ubuntu:latest",
		Features: map[string]interface{}{
			featureRef: map[string]interface{}{
				"version": "latest",
			},
		},
	}

	err := RunUp(opts)
	if err != nil {
		t.Fatalf("RunUp with features failed: %v", err)
	}

	// Verify command sequence includes build and run commands using the layered features image
	buildCallFound := false
	runCallFound := false

	for _, call := range runner.commandsRun {
		if len(call) > 1 && call[1] == "build" {
			buildCallFound = true
			// Check image tagging target
			tagFound := false
			for i, arg := range call {
				if arg == "-t" && i+1 < len(call) && call[i+1] == "non-existent-container-features" {
					tagFound = true
				}
			}
			if !tagFound {
				t.Errorf("Expected podman build to tag target image 'non-existent-container-features', got: %v", call)
			}
		}

		if len(call) > 1 && call[1] == "run" {
			runCallFound = true
			// Ensure it runs with the layered features image
			imageFound := false
			for _, arg := range call {
				if arg == "non-existent-container-features" {
					imageFound = true
				}
			}
			if !imageFound {
				t.Errorf("Expected podman run to use 'non-existent-container-features' image, got: %v", call)
			}
		}
	}

	if !buildCallFound {
		t.Fatal("Expected podman build command call, none found")
	}
	if !runCallFound {
		t.Fatal("Expected podman run command call, none found")
	}
}

func TestRunUpWorkflowComposeWithFeatures(t *testing.T) {
	// Create dummy tarball bytes representing feature payload
	var tarballBuf bytes.Buffer
	gw := gzip.NewWriter(&tarballBuf)
	tw := tar.NewWriter(gw)

	// Add install.sh
	installShContent := "echo 'installing feature'"
	hdr1 := &tar.Header{
		Name: "install.sh",
		Mode: 0755,
		Size: int64(len(installShContent)),
	}
	tw.WriteHeader(hdr1)
	tw.Write([]byte(installShContent))

	// Add devcontainer-feature.json
	metaContent := `{"id": "git", "installsAfter": []}`
	hdr2 := &tar.Header{
		Name: "devcontainer-feature.json",
		Mode: 0644,
		Size: int64(len(metaContent)),
	}
	tw.WriteHeader(hdr2)
	tw.Write([]byte(metaContent))

	tw.Close()
	gw.Close()

	tarballBytes := tarballBuf.Bytes()

	// Start local mock registry server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Mock token auth challenge
		if r.URL.Path != "/token" && r.Header.Get("Authorization") == "" {
			w.Header().Set("Www-Authenticate", `Bearer realm="http://`+r.Host+`/token",service="mock-registry",scope="repository:devcontainers/features/git:pull"`)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if r.URL.Path == "/token" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"token": "mock-access-token"}`))
			return
		}

		if strings.Contains(r.URL.Path, "/manifests/") {
			w.Header().Set("Content-Type", features.OciManifestMediaType)
			w.Write([]byte(`{
				"schemaVersion": 2,
				"mediaType": "application/vnd.oci.image.manifest.v1+json",
				"layers": [
					{
						"mediaType": "application/vnd.devcontainers.layer.v1+tar+gzip",
						"digest": "sha256:abc123digest",
						"size": 1234
					}
				]
			}`))
			return
		}

		if strings.Contains(r.URL.Path, "/blobs/") {
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write(tarballBytes)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	urlHost := strings.TrimPrefix(server.URL, "http://")

	os.Setenv("SSH_AUTH_SOCK", "")
	oldDetector := gpgAgentSocketDetector
	gpgAgentSocketDetector = func() string { return "" }
	defer func() { gpgAgentSocketDetector = oldDetector }()

	runner := &mockDockerRunner{}
	dockerCLI := docker.NewCLI("podman", "podman-compose", runner.Run)

	featureRef := urlHost + "/devcontainers/features/git:1"
	opts := UpOptions{
		DockerCLI:         dockerCLI,
		WorkspaceFolder:   "/my-workspace",
		ContainerName:     "my-container",
		BaseImage:         "ubuntu:latest",
		DockerComposeFile: "docker-compose.yml",
		Service:           "db",
		ConfigPath:        "/my-workspace/devcontainer.json",
		Features: map[string]interface{}{
			featureRef: map[string]interface{}{
				"version": "latest",
			},
		},
	}

	err := RunUp(opts)
	if err != nil {
		t.Fatalf("RunUp with features failed: %v", err)
	}

	// Verify build features layered image was triggered
	buildCallFound := false
	composeUpCallFound := false

	for _, call := range runner.commandsRun {
		if len(call) > 1 && call[1] == "build" {
			buildCallFound = true
			tagFound := false
			for i, arg := range call {
				if arg == "-t" && i+1 < len(call) && call[i+1] == "my-container-features" {
					tagFound = true
				}
			}
			if !tagFound {
				t.Errorf("Expected build command to tag target image 'my-container-features', got: %v", call)
			}
		}

		isComposeUp := false
		for _, arg := range call {
			if arg == "up" {
				isComposeUp = true
				break
			}
		}

		if isComposeUp {
			// Podman compose up args: [podman-compose -f /my-workspace/docker-compose.yml -f /tmp/... -p my-workspace up -d]
			composeUpCallFound = true
			// Check that two -f arguments are supplied
			fCount := 0
			for _, arg := range call {
				if arg == "-f" {
					fCount++
				}
			}
			if fCount != 2 {
				t.Errorf("Expected compose up command to use 2 -f flags, got: %d in %v", fCount, call)
			}
		}
	}

	if !buildCallFound {
		t.Fatal("Expected podman build command call, none found")
	}
	if !composeUpCallFound {
		t.Fatal("Expected compose up command call, none found")
	}
}
