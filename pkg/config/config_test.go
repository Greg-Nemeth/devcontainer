package config

import (
	"reflect"
	"testing"
)

func TestStripComments(t *testing.T) {
	input := `{
		// A single line comment
		"name": "Ubuntu", /* block comment */
		"image": "mcr.microsoft.com/devcontainers/base:ubuntu",
		"forwardPorts": [
			8080, // port for web server
			9000 /* port for debugger */
		]
	}`

	expected := `{
		
		"name": "Ubuntu", 
		"image": "mcr.microsoft.com/devcontainers/base:ubuntu",
		"forwardPorts": [
			8080, 
			9000 
		]
	}`

	got := stripComments(input)
	// Compare without whitespace/newlines to avoid formatting mismatches
	cleanGot := removeWhitespace(got)
	cleanExpected := removeWhitespace(expected)

	if cleanGot != cleanExpected {
		t.Errorf("stripComments failed.\nGot: %q\nWant: %q", got, expected)
	}
}

func removeWhitespace(s string) string {
	var res []rune
	for _, r := range s {
		if r != ' ' && r != '\t' && r != '\n' && r != '\r' {
			res = append(res, r)
		}
	}
	return string(res)
}

func TestParseConfig(t *testing.T) {
	jsoncStr := `{
		// Base devcontainer config
		"name": "Go Devcontainer",
		"image": "golang:1.20",
		"forwardPorts": [8080],
		"features": {
			"ghcr.io/devcontainers/features/git:1": {
				"version": "latest"
			}
		}
	}`

	cfg, err := Parse(jsoncStr)
	if err != nil {
		t.Fatalf("Parse returned unexpected error: %v", err)
	}

	if cfg.Name != "Go Devcontainer" {
		t.Errorf("Expected Name 'Go Devcontainer', got %q", cfg.Name)
	}
	if cfg.Image != "golang:1.20" {
		t.Errorf("Expected Image 'golang:1.20', got %q", cfg.Image)
	}
	if !reflect.DeepEqual(cfg.ForwardPorts, []int{8080}) {
		t.Errorf("Expected ForwardPorts [8080], got %v", cfg.ForwardPorts)
	}
}

func TestMergeConfigs(t *testing.T) {
	base := &DevContainerConfig{
		Name:         "Base",
		Image:        "ubuntu",
		ForwardPorts: []int{8080},
		Features: map[string]interface{}{
			"git": "latest",
		},
	}

	override := &DevContainerConfig{
		Name:         "Override",
		ForwardPorts: []int{9000},
		Features: map[string]interface{}{
			"docker-in-docker": "latest",
		},
	}

	merged := Merge(base, override)

	if merged.Name != "Override" {
		t.Errorf("Expected Name 'Override' after merge, got %q", merged.Name)
	}
	if merged.Image != "ubuntu" {
		t.Errorf("Expected Image 'ubuntu' from base, got %q", merged.Image)
	}
	// forwardPorts override
	if !reflect.DeepEqual(merged.ForwardPorts, []int{9000}) {
		t.Errorf("Expected ForwardPorts [9000], got %v", merged.ForwardPorts)
	}
	// features merged
	if merged.Features["git"] != "latest" || merged.Features["docker-in-docker"] != "latest" {
		t.Errorf("Expected features merged, got %v", merged.Features)
	}
}
