package docker

import (
	"reflect"
	"strings"
	"testing"
)

func TestGenerateComposeOverride(t *testing.T) {
	opts := ComposeOverrideOptions{
		Service: "web-app",
		Image:   "my-built-features-image:latest",
		Mounts: []string{
			"type=bind,source=/home/user/app,target=/workspace",
		},
		Env: map[string]string{
			"DB_HOST": "localhost",
			"DEBUG":   "true",
		},
	}

	yamlContent, err := GenerateComposeOverride(opts)
	if err != nil {
		t.Fatalf("GenerateComposeOverride returned unexpected error: %v", err)
	}

	// Verify target service
	if !strings.Contains(yamlContent, "web-app:") {
		t.Errorf("Expected YAML to contain 'web-app:', got:\n%s", yamlContent)
	}

	// Verify image override
	if !strings.Contains(yamlContent, "image: my-built-features-image:latest") {
		t.Errorf("Expected YAML to contain image override, got:\n%s", yamlContent)
	}

	// Verify mounts
	if !strings.Contains(yamlContent, "- type=bind,source=/home/user/app,target=/workspace") {
		t.Errorf("Expected YAML to contain mounts, got:\n%s", yamlContent)
	}

	// Verify env variables
	if !strings.Contains(yamlContent, "- DB_HOST=localhost") || !strings.Contains(yamlContent, "- DEBUG=true") {
		t.Errorf("Expected YAML to contain environment list, got:\n%s", yamlContent)
	}
}

func TestParseComposeConfigMapArgs(t *testing.T) {
	yamlInput := `
services:
  web-app:
    image: node:18
    build:
      context: ./src
      dockerfile: Custom.Dockerfile
      args:
        NODE_ENV: production
        APP_PORT: "8080"
`

	info, err := ParseComposeConfig(yamlInput, "web-app")
	if err != nil {
		t.Fatalf("ParseComposeConfig failed: %v", err)
	}

	if info.Image != "node:18" {
		t.Errorf("Expected image 'node:18', got %q", info.Image)
	}
	if info.Context != "./src" {
		t.Errorf("Expected context './src', got %q", info.Context)
	}
	if info.Dockerfile != "Custom.Dockerfile" {
		t.Errorf("Expected dockerfile 'Custom.Dockerfile', got %q", info.Dockerfile)
	}

	expectedArgs := map[string]string{
		"NODE_ENV": "production",
		"APP_PORT": "8080",
	}
	if !reflect.DeepEqual(info.Args, expectedArgs) {
		t.Errorf("Expected args %v, got %v", expectedArgs, info.Args)
	}
}

func TestParseComposeConfigListArgs(t *testing.T) {
	// Podman compose config format using list array for build arguments
	yamlInput := `
services:
  web-app:
    build:
      context: .
      args:
        - NODE_ENV=production
        - APP_PORT=8080
`

	info, err := ParseComposeConfig(yamlInput, "web-app")
	if err != nil {
		t.Fatalf("ParseComposeConfig failed: %v", err)
	}

	expectedArgs := map[string]string{
		"NODE_ENV": "production",
		"APP_PORT": "8080",
	}
	if !reflect.DeepEqual(info.Args, expectedArgs) {
		t.Errorf("Expected args %v, got %v", expectedArgs, info.Args)
	}
}
