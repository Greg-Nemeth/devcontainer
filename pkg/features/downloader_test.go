package features

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseBearerChallenge(t *testing.T) {
	headerVal := `Bearer realm="https://ghcr.io/token",service="ghcr.io",scope="repository:devcontainers/features/git:pull"`
	realm, service, scope, err := parseBearerChallenge(headerVal)
	if err != nil {
		t.Fatalf("parseBearerChallenge returned unexpected error: %v", err)
	}

	if realm != "https://ghcr.io/token" {
		t.Errorf("Expected realm 'https://ghcr.io/token', got %q", realm)
	}
	if service != "ghcr.io" {
		t.Errorf("Expected service 'ghcr.io', got %q", service)
	}
	if scope != "repository:devcontainers/features/git:pull" {
		t.Errorf("Expected scope 'repository:devcontainers/features/git:pull', got %q", scope)
	}
}

func TestExtractTarGz(t *testing.T) {
	// Create a dummy tar.gz payload in memory
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	// Add a test file
	fileContent := "echo 'installing feature'"
	hdr := &tar.Header{
		Name: "install.sh",
		Mode: 0755,
		Size: int64(len(fileContent)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatalf("Failed to write tar header: %v", err)
	}
	if _, err := tw.Write([]byte(fileContent)); err != nil {
		t.Fatalf("Failed to write tar content: %v", err)
	}

	tw.Close()
	gw.Close()

	// Extract it to a temporary directory
	tmpDir, err := os.MkdirTemp("", "dc-extract-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	err = ExtractTarGz(bytes.NewReader(buf.Bytes()), tmpDir)
	if err != nil {
		t.Fatalf("ExtractTarGz failed: %v", err)
	}

	// Verify the file was extracted correctly
	extractedFilePath := filepath.Join(tmpDir, "install.sh")
	data, err := os.ReadFile(extractedFilePath)
	if err != nil {
		t.Fatalf("Failed to read extracted file: %v", err)
	}

	if string(data) != fileContent {
		t.Errorf("Extracted file content mismatch.\nGot:  %q\nWant: %q", string(data), fileContent)
	}
}

func TestOCIClientFetchManifestMock(t *testing.T) {
	// Start a local mock server representing the registry v2 endpoint
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Mock token auth challenge
		if r.URL.Path != "/token" && r.Header.Get("Authorization") == "" {
			w.Header().Set("Www-Authenticate", `Bearer realm="http://`+r.Host+`/token",service="mock-registry",scope="repository:foo/bar:pull"`)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if r.URL.Path == "/token" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"token": "mock-access-token"}`))
			return
		}

		if strings.Contains(r.URL.Path, "/manifests/") {
			w.Header().Set("Content-Type", OciManifestMediaType)
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

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	// Parse custom URL format compatible with mock server registry
	// URL of mock server is parsed into host
	urlHost := strings.TrimPrefix(server.URL, "http://")
	ref := FeatureRef{
		Registry:  urlHost,
		Namespace: "foo",
		ID:        "bar",
		Version:   "1",
	}

	client := NewOCIClient(server.Client())
	manifest, err := client.FetchManifest(ref)
	if err != nil {
		t.Fatalf("FetchManifest failed: %v", err)
	}

	if len(manifest.Layers) != 1 {
		t.Fatalf("Expected 1 layer, got %d", len(manifest.Layers))
	}

	if manifest.Layers[0].Digest != "sha256:abc123digest" {
		t.Errorf("Expected layer digest 'sha256:abc123digest', got %q", manifest.Layers[0].Digest)
	}
}
