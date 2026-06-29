package cli

import (
	"strings"
	"testing"
)

func TestGPGAgentSocketDetection(t *testing.T) {
	socket := detectGPGAgentSocket()
	t.Logf("Detected GPG socket: %q", socket)
}

func TestInjectHeadlessServerScript(t *testing.T) {
	// Test generating download/setup script for openvscode server
	script, err := getBootstrapperScript("openvscode")
	if err != nil {
		t.Fatalf("Failed to generate bootstrapper: %v", err)
	}

	if !strings.Contains(script, "wget") && !strings.Contains(script, "curl") {
		t.Errorf("Expected script to use wget or curl, got:\n%s", script)
	}

	if !strings.Contains(script, "tar") {
		t.Errorf("Expected script to unpack tarball, got:\n%s", script)
	}
}
