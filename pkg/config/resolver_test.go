package config

import (
	"reflect"
	"testing"
)

func TestParseImageMetadata(t *testing.T) {
	// Sample output from docker inspect image containing the devcontainer.metadata label
	inspectJSON := `[
		{
			"Id": "sha256:12345",
			"Config": {
				"Labels": {
					"devcontainer.metadata": "[{\"id\": \"myFeature\", \"containerEnv\": {\"FEAT_VAR\": \"yes\"}}, {\"postCreateCommand\": \"echo hello\"}]"
				}
			}
		}
	]`

	metadata, err := ParseImageMetadata([]byte(inspectJSON))
	if err != nil {
		t.Fatalf("ParseImageMetadata returned unexpected error: %v", err)
	}

	if len(metadata) != 2 {
		t.Fatalf("Expected 2 metadata entries, got %d", len(metadata))
	}

	if env, ok := metadata[0].ContainerEnv["FEAT_VAR"]; !ok || env != "yes" {
		t.Errorf("Expected ContainerEnv FEAT_VAR to be 'yes', got %q", env)
	}

	if cmd, ok := metadata[1].PostCreateCommand.(string); !ok || cmd != "echo hello" {
		t.Errorf("Expected PostCreateCommand to be 'echo hello', got %v", metadata[1].PostCreateCommand)
	}
}

func TestMergeConfigurationWithMetadata(t *testing.T) {
	base := &DevContainerConfig{
		Name:  "BaseConfig",
		Image: "ubuntu",
		ContainerEnv: map[string]string{
			"VAR1": "base_val",
		},
	}

	meta1 := DevContainerConfig{
		ContainerEnv: map[string]string{
			"VAR1": "meta1_val",
			"VAR2": "meta2_val",
		},
		PostCreateCommand: "setup-feature",
	}

	meta2 := DevContainerConfig{
		PostCreateCommand: []interface{}{"echo", "done"},
	}

	// Merge configuration with metadata entries
	merged := MergeWithMetadata(base, []DevContainerConfig{meta1, meta2})

	// Check containerEnv overrides (meta overrides base, later metadata overrides earlier metadata)
	expectedEnv := map[string]string{
		"VAR1": "meta1_val",
		"VAR2": "meta2_val",
	}
	if !reflect.DeepEqual(merged.ContainerEnv, expectedEnv) {
		t.Errorf("Expected ContainerEnv %v, got %v", expectedEnv, merged.ContainerEnv)
	}

	// Verify lifecycle commands are accumulated as a slice of commands
	// In the merged config, the lifecycle hooks from image metadata are collected.
	// For example, merged.PostCreateCommands should contain both commands.
	if len(merged.PostCreateCommands) != 2 {
		t.Fatalf("Expected 2 PostCreateCommands, got %d", len(merged.PostCreateCommands))
	}
	if merged.PostCreateCommands[0] != "setup-feature" {
		t.Errorf("Expected first PostCreateCommand to be 'setup-feature', got %v", merged.PostCreateCommands[0])
	}
}
