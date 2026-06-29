package config

import (
	"encoding/json"
	"strings"
)

type DevContainerConfig struct {
	Name                 string                 `json:"name,omitempty"`
	Image                string                 `json:"image,omitempty"`
	Dockerfile           string                 `json:"dockerFile,omitempty"`
	DockerComposeFile    interface{}            `json:"dockerComposeFile,omitempty"` // string or []string
	Service              string                 `json:"service,omitempty"`
	WorkspaceFolder      string                 `json:"workspaceFolder,omitempty"`
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
