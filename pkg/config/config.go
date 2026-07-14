package config

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/devcontainers/dc/pkg/varsub"
)

type DevContainerConfig struct {
	Name                 string                 `json:"name,omitempty"`
	Image                string                 `json:"image,omitempty"`
	Dockerfile           string                 `json:"dockerFile,omitempty"`
	DockerComposeFile    interface{}            `json:"dockerComposeFile,omitempty"` // string or []string
	Service              string                 `json:"service,omitempty"`
	WorkspaceFolder      string                 `json:"workspaceFolder,omitempty"`
	WorkspaceMount       string                 `json:"workspaceMount,omitempty"`
	ShutdownAction       string                 `json:"shutdownAction,omitempty"`
	ForwardPorts         []int                  `json:"forwardPorts,omitempty"`
	PortsAttributes      map[string]interface{} `json:"portsAttributes,omitempty"`
	Features             map[string]interface{} `json:"features,omitempty"`
	OverrideCommand      *bool                  `json:"overrideCommand,omitempty"`
	Mounts               []interface{}          `json:"mounts,omitempty"`
	ContainerEnv         map[string]string      `json:"containerEnv,omitempty"`
	RemoteEnv            map[string]string      `json:"remoteEnv,omitempty"`
	PostCreateCommand    interface{}            `json:"postCreateCommand,omitempty"`
	UpdateContentCommand interface{}            `json:"updateContentCommand,omitempty"`
	PostStartCommand     interface{}            `json:"postStartCommand,omitempty"`
	PostAttachCommand    interface{}            `json:"postAttachCommand,omitempty"`
	OnCreateCommand      interface{}            `json:"onCreateCommand,omitempty"`

	// Accumulated/merged fields from image metadata
	PostCreateCommands    []interface{} `json:"postCreateCommands,omitempty"`
	UpdateContentCommands []interface{} `json:"updateContentCommands,omitempty"`
	PostStartCommands     []interface{} `json:"postStartCommands,omitempty"`
	PostAttachCommands    []interface{} `json:"postAttachCommands,omitempty"`
	OnCreateCommands      []interface{} `json:"onCreateCommands,omitempty"`

	Customizations map[string]interface{} `json:"customizations,omitempty"`
	Entrypoint     string                 `json:"entrypoint,omitempty"`
	Entrypoints    []string               `json:"entrypoints,omitempty"`
	RemoteUser     string                 `json:"remoteUser,omitempty"`
	ContainerUser  string                 `json:"containerUser,omitempty"`
	UserEnvProbe   string                 `json:"userEnvProbe,omitempty"`
	Privileged     bool                   `json:"privileged,omitempty"`
	CapAdd         []string               `json:"capAdd,omitempty"`
	SecurityOpt    []string               `json:"securityOpt,omitempty"`
	Init           bool                   `json:"init,omitempty"`
}

func stripComments(input string) string {
	var result strings.Builder
	state := 0 // 0: Normal, 1: String, 2: Escape, 3: LineComment, 4: BlockComment
	runes := []rune(input)
	n := len(runes)

	for i := 0; i < n; i++ {
		r := runes[i]
		switch state {
		case 0: // Normal
			if r == '"' {
				state = 1
				result.WriteRune(r)
			} else if r == '/' && i+1 < n && runes[i+1] == '/' {
				state = 3
				i++
			} else if r == '/' && i+1 < n && runes[i+1] == '*' {
				state = 4
				i++
			} else {
				result.WriteRune(r)
			}
		case 1: // String
			if r == '\\' {
				state = 2
				result.WriteRune(r)
			} else if r == '"' {
				state = 0
				result.WriteRune(r)
			} else {
				result.WriteRune(r)
			}
		case 2: // Escape in String
			state = 1
			result.WriteRune(r)
		case 3: // LineComment
			if r == '\n' || r == '\r' {
				state = 0
				result.WriteRune(r)
			}
		case 4: // BlockComment
			if r == '*' && i+1 < n && runes[i+1] == '/' {
				state = 0
				i++
			}
		}
	}
	return result.String()
}

func Parse(jsoncContent string) (*DevContainerConfig, error) {
	cleanJSON := stripComments(jsoncContent)
	var cfg DevContainerConfig
	if err := json.Unmarshal([]byte(cleanJSON), &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func Merge(base, override *DevContainerConfig) *DevContainerConfig {
	if base == nil {
		return override
	}
	if override == nil {
		return base
	}

	merged := *base

	if override.Name != "" {
		merged.Name = override.Name
	}
	if override.Image != "" {
		merged.Image = override.Image
	}
	if override.Dockerfile != "" {
		merged.Dockerfile = override.Dockerfile
	}
	if override.DockerComposeFile != nil {
		merged.DockerComposeFile = override.DockerComposeFile
	}
	if override.Service != "" {
		merged.Service = override.Service
	}
	if override.WorkspaceFolder != "" {
		merged.WorkspaceFolder = override.WorkspaceFolder
	}
	if override.WorkspaceMount != "" {
		merged.WorkspaceMount = override.WorkspaceMount
	}
	if override.ShutdownAction != "" {
		merged.ShutdownAction = override.ShutdownAction
	}
	if override.ForwardPorts != nil {
		merged.ForwardPorts = override.ForwardPorts
	}
	if override.PortsAttributes != nil {
		if merged.PortsAttributes == nil {
			merged.PortsAttributes = make(map[string]interface{})
		}
		for k, v := range override.PortsAttributes {
			merged.PortsAttributes[k] = v
		}
	}
	if override.Features != nil {
		if merged.Features == nil {
			merged.Features = make(map[string]interface{})
		}
		for k, v := range override.Features {
			merged.Features[k] = v
		}
	}
	if override.OverrideCommand != nil {
		merged.OverrideCommand = override.OverrideCommand
	}
	if override.Mounts != nil {
		merged.Mounts = override.Mounts
	}
	if override.ContainerEnv != nil {
		if merged.ContainerEnv == nil {
			merged.ContainerEnv = make(map[string]string)
		}
		for k, v := range override.ContainerEnv {
			merged.ContainerEnv[k] = v
		}
	}
	if override.RemoteEnv != nil {
		if merged.RemoteEnv == nil {
			merged.RemoteEnv = make(map[string]string)
		}
		for k, v := range override.RemoteEnv {
			merged.RemoteEnv[k] = v
		}
	}
	if override.PostCreateCommand != nil {
		merged.PostCreateCommand = override.PostCreateCommand
	}
	if override.UpdateContentCommand != nil {
		merged.UpdateContentCommand = override.UpdateContentCommand
	}
	if override.PostStartCommand != nil {
		merged.PostStartCommand = override.PostStartCommand
	}
	if override.PostAttachCommand != nil {
		merged.PostAttachCommand = override.PostAttachCommand
	}
	if override.OnCreateCommand != nil {
		merged.OnCreateCommand = override.OnCreateCommand
	}

	return &merged
}

func (cfg *DevContainerConfig) SubstituteVariables(wsFolder string, cfgPath string) {
	// Build Env map
	envMap := make(map[string]string)
	for _, env := range os.Environ() {
		kv := strings.SplitN(env, "=", 2)
		if len(kv) == 2 {
			envMap[kv[0]] = kv[1]
		}
	}

	containerWorkspaceFolder := cfg.WorkspaceFolder
	if containerWorkspaceFolder == "" {
		containerWorkspaceFolder = "/workspace" // default fallback
	}

	ctx := varsub.SubstitutionContext{
		Platform:                 "linux",
		ConfigFile:               cfgPath,
		LocalWorkspaceFolder:     wsFolder,
		ContainerWorkspaceFolder: containerWorkspaceFolder,
		Env:                      envMap,
	}

	// Substitute simple string fields
	if cfg.Image != "" {
		if s, ok := varsub.Substitute(ctx, cfg.Image).(string); ok {
			cfg.Image = s
		}
	}
	if cfg.Dockerfile != "" {
		if s, ok := varsub.Substitute(ctx, cfg.Dockerfile).(string); ok {
			cfg.Dockerfile = s
		}
	}
	if cfg.WorkspaceFolder != "" {
		if s, ok := varsub.Substitute(ctx, cfg.WorkspaceFolder).(string); ok {
			cfg.WorkspaceFolder = s
		}
	}
	if cfg.WorkspaceMount != "" {
		if s, ok := varsub.Substitute(ctx, cfg.WorkspaceMount).(string); ok {
			cfg.WorkspaceMount = s
		}
	}
	if cfg.RemoteUser != "" {
		if s, ok := varsub.Substitute(ctx, cfg.RemoteUser).(string); ok {
			cfg.RemoteUser = s
		}
	}
	if cfg.ContainerUser != "" {
		if s, ok := varsub.Substitute(ctx, cfg.ContainerUser).(string); ok {
			cfg.ContainerUser = s
		}
	}

	// Substitute OnCreateCommand, PostCreateCommand, etc.
	if cfg.OnCreateCommand != nil {
		cfg.OnCreateCommand = varsub.Substitute(ctx, cfg.OnCreateCommand)
	}
	if cfg.PostCreateCommand != nil {
		cfg.PostCreateCommand = varsub.Substitute(ctx, cfg.PostCreateCommand)
	}
	if cfg.UpdateContentCommand != nil {
		cfg.UpdateContentCommand = varsub.Substitute(ctx, cfg.UpdateContentCommand)
	}
	if cfg.PostStartCommand != nil {
		cfg.PostStartCommand = varsub.Substitute(ctx, cfg.PostStartCommand)
	}
	if cfg.PostAttachCommand != nil {
		cfg.PostAttachCommand = varsub.Substitute(ctx, cfg.PostAttachCommand)
	}

	// Substitute ContainerEnv and RemoteEnv maps
	if cfg.ContainerEnv != nil {
		newEnv := make(map[string]string)
		for k, v := range cfg.ContainerEnv {
			if s, ok := varsub.Substitute(ctx, v).(string); ok {
				newEnv[k] = s
			} else {
				newEnv[k] = v
			}
		}
		cfg.ContainerEnv = newEnv
	}
	if cfg.RemoteEnv != nil {
		newEnv := make(map[string]string)
		for k, v := range cfg.RemoteEnv {
			if s, ok := varsub.Substitute(ctx, v).(string); ok {
				newEnv[k] = s
			} else {
				newEnv[k] = v
			}
		}
		cfg.RemoteEnv = newEnv
	}

	// Substitute Mounts slice
	if cfg.Mounts != nil {
		newMounts := make([]interface{}, len(cfg.Mounts))
		for i, m := range cfg.Mounts {
			newMounts[i] = varsub.Substitute(ctx, m)
		}
		cfg.Mounts = newMounts
	}
}
