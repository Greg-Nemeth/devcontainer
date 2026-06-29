package config

import (
	"encoding/json"
	"fmt"
)

type ImageInspectResult struct {
	Config struct {
		Labels map[string]string `json:"Labels"`
	} `json:"Config"`
}

const ImageMetadataLabel = "devcontainer.metadata"

func ParseImageMetadata(inspectOutput []byte) ([]DevContainerConfig, error) {
	var results []ImageInspectResult
	if err := json.Unmarshal(inspectOutput, &results); err != nil {
		return nil, fmt.Errorf("failed to parse image inspect output: %w", err)
	}

	if len(results) == 0 {
		return nil, nil
	}

	labelVal, exists := results[0].Config.Labels[ImageMetadataLabel]
	if !exists || labelVal == "" {
		return nil, nil
	}

	// Try parsing as array
	var list []DevContainerConfig
	if err := json.Unmarshal([]byte(labelVal), &list); err == nil {
		return list, nil
	}

	// Try parsing as single object
	var single DevContainerConfig
	if err := json.Unmarshal([]byte(labelVal), &single); err == nil {
		return []DevContainerConfig{single}, nil
	}

	return nil, fmt.Errorf("failed to parse devcontainer.metadata label value: %s", labelVal)
}

func MergeWithMetadata(base *DevContainerConfig, metadata []DevContainerConfig) *DevContainerConfig {
	if base == nil {
		base = &DevContainerConfig{}
	}

	merged := *base

	// We append base to the end of the metadata slice to process in order:
	// base is processed first, then metadata from first to last (later overrides earlier).
	// But note that for lifecycle command accumulation, base commands should be run,
	// and they represent the final user-defined commands.
	allConfigs := append([]DevContainerConfig{}, metadata...)

	// Container Env and Remote Env merging (later overrides earlier)
	if merged.ContainerEnv == nil {
		merged.ContainerEnv = make(map[string]string)
	}
	for _, cfg := range allConfigs {
		for k, v := range cfg.ContainerEnv {
			merged.ContainerEnv[k] = v
		}
	}

	if merged.RemoteEnv == nil {
		merged.RemoteEnv = make(map[string]string)
	}
	for _, cfg := range allConfigs {
		for k, v := range cfg.RemoteEnv {
			merged.RemoteEnv[k] = v
		}
	}

	// Accumulate lifecycle hooks
	// In upstream, hooks are collected from all metadata entries and base config
	merged.OnCreateCommands = collectLifecycleCommands(base.OnCreateCommand, allConfigs, "OnCreateCommand")
	merged.UpdateContentCommands = collectLifecycleCommands(base.UpdateContentCommand, allConfigs, "UpdateContentCommand")
	merged.PostCreateCommands = collectLifecycleCommands(base.PostCreateCommand, allConfigs, "PostCreateCommand")
	merged.PostStartCommands = collectLifecycleCommands(base.PostStartCommand, allConfigs, "PostStartCommand")
	merged.PostAttachCommands = collectLifecycleCommands(base.PostAttachCommand, allConfigs, "PostAttachCommand")

	// Init, Privileged (true if any is true)
	merged.Init = base.Init
	merged.Privileged = base.Privileged
	for _, cfg := range allConfigs {
		if cfg.Init {
			merged.Init = true
		}
		if cfg.Privileged {
			merged.Privileged = true
		}
	}

	// CapAdd, SecurityOpt union
	capAddMap := make(map[string]bool)
	for _, c := range base.CapAdd {
		capAddMap[c] = true
	}
	for _, cfg := range allConfigs {
		for _, c := range cfg.CapAdd {
			capAddMap[c] = true
		}
	}
	merged.CapAdd = nil
	for c := range capAddMap {
		merged.CapAdd = append(merged.CapAdd, c)
	}

	secOptMap := make(map[string]bool)
	for _, s := range base.SecurityOpt {
		secOptMap[s] = true
	}
	for _, cfg := range allConfigs {
		for _, s := range cfg.SecurityOpt {
			secOptMap[s] = true
		}
	}
	merged.SecurityOpt = nil
	for s := range secOptMap {
		merged.SecurityOpt = append(merged.SecurityOpt, s)
	}

	// Entrypoints
	var entrypoints []string
	if base.Entrypoint != "" {
		entrypoints = append(entrypoints, base.Entrypoint)
	}
	for _, cfg := range allConfigs {
		if cfg.Entrypoint != "" {
			entrypoints = append(entrypoints, cfg.Entrypoint)
		}
	}
	merged.Entrypoints = entrypoints

	// Host parameters (like remoteUser, containerUser, userEnvProbe)
	// Last non-empty wins. Scan backwards from metadata, then fallback to base.
	for i := len(metadata) - 1; i >= 0; i-- {
		cfg := metadata[i]
		if merged.RemoteUser == "" && cfg.RemoteUser != "" {
			merged.RemoteUser = cfg.RemoteUser
		}
		if merged.ContainerUser == "" && cfg.ContainerUser != "" {
			merged.ContainerUser = cfg.ContainerUser
		}
		if merged.UserEnvProbe == "" && cfg.UserEnvProbe != "" {
			merged.UserEnvProbe = cfg.UserEnvProbe
		}
	}

	return &merged
}

func collectLifecycleCommands(baseCommand interface{}, metadata []DevContainerConfig, fieldName string) []interface{} {
	var collected []interface{}

	// Add commands from metadata first
	for _, cfg := range metadata {
		var cmd interface{}
		switch fieldName {
		case "OnCreateCommand":
			cmd = cfg.OnCreateCommand
		case "UpdateContentCommand":
			cmd = cfg.UpdateContentCommand
		case "PostCreateCommand":
			cmd = cfg.PostCreateCommand
		case "PostStartCommand":
			cmd = cfg.PostStartCommand
		case "PostAttachCommand":
			cmd = cfg.PostAttachCommand
		}

		if cmd != nil && !isCommandEmpty(cmd) {
			collected = append(collected, cmd)
		}
	}

	// Add base command if set
	if baseCommand != nil && !isCommandEmpty(baseCommand) {
		collected = append(collected, baseCommand)
	}

	return collected
}

func isCommandEmpty(cmd interface{}) bool {
	if s, ok := cmd.(string); ok {
		return s == ""
	}
	if slice, ok := cmd.([]interface{}); ok {
		return len(slice) == 0
	}
	return false
}
