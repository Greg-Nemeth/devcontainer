package templates

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseTemplateRef(t *testing.T) {
	tests := []struct {
		input    string
		expected TemplateRef
		wantErr  bool
	}{
		{
			input: "ghcr.io/devcontainers/templates/go:1",
			expected: TemplateRef{
				Registry:  "ghcr.io",
				Namespace: "devcontainers/templates",
				ID:        "go",
				Version:   "1",
			},
			wantErr: false,
		},
		{
			input:    "invalid-template-ref",
			expected: TemplateRef{},
			wantErr:  true,
		},
	}

	for _, tc := range tests {
		got, err := ParseTemplateRef(tc.input)
		if (err != nil) != tc.wantErr {
			t.Errorf("ParseTemplateRef(%q) err = %v; wantErr = %v", tc.input, err, tc.wantErr)
			continue
		}
		if !tc.wantErr && !reflect.DeepEqual(got, tc.expected) {
			t.Errorf("ParseTemplateRef(%q) = %+v; want %+v", tc.input, got, tc.expected)
		}
	}
}

func TestApplyTemplate(t *testing.T) {
	// Create a temp source directory mimicking the unpacked template
	srcDir, err := os.MkdirTemp("", "dc-template-src-*")
	if err != nil {
		t.Fatalf("Failed to create temp src dir: %v", err)
	}
	defer os.RemoveAll(srcDir)

	// Create a subfolder and files with option placeholders
	os.MkdirAll(filepath.Join(srcDir, ".devcontainer"), 0755)
	err = os.WriteFile(
		filepath.Join(srcDir, ".devcontainer", "devcontainer.json"),
		[]byte(`{ "name": "Go Project", "image": "mcr.microsoft.com/devcontainers/go:${templateOption:goVersion}" }`),
		0644,
	)
	if err != nil {
		t.Fatalf("Failed to write mock devcontainer.json: %v", err)
	}

	err = os.WriteFile(
		filepath.Join(srcDir, "README.md"),
		[]byte("# Welcome to ${templateOption:projectName}"),
		0644,
	)
	if err != nil {
		t.Fatalf("Failed to write mock README.md: %v", err)
	}

	// Create a target directory to apply the template
	dstDir, err := os.MkdirTemp("", "dc-template-dst-*")
	if err != nil {
		t.Fatalf("Failed to create temp dst dir: %v", err)
	}
	defer os.RemoveAll(dstDir)

	options := map[string]string{
		"goVersion":   "1.21",
		"projectName": "My Awesome App",
	}

	err = ApplyTemplate(srcDir, dstDir, options)
	if err != nil {
		t.Fatalf("ApplyTemplate failed: %v", err)
	}

	// Verify devcontainer.json substitution
	devcontainerData, err := os.ReadFile(filepath.Join(dstDir, ".devcontainer", "devcontainer.json"))
	if err != nil {
		t.Fatalf("Failed to read applied devcontainer.json: %v", err)
	}
	expectedJSON := `{ "name": "Go Project", "image": "mcr.microsoft.com/devcontainers/go:1.21" }`
	if string(devcontainerData) != expectedJSON {
		t.Errorf("Expected devcontainer.json:\n%s\nGot:\n%s", expectedJSON, string(devcontainerData))
	}

	// Verify README.md substitution
	readmeData, err := os.ReadFile(filepath.Join(dstDir, "README.md"))
	if err != nil {
		t.Fatalf("Failed to read applied README.md: %v", err)
	}
	expectedReadme := "# Welcome to My Awesome App"
	if string(readmeData) != expectedReadme {
		t.Errorf("Expected README.md:\n%s\nGot:\n%s", expectedReadme, string(readmeData))
	}
}
