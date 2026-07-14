package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/devcontainers/dc/pkg/config"
	"github.com/devcontainers/dc/pkg/docker"
	"github.com/devcontainers/dc/pkg/features"
	"github.com/devcontainers/dc/pkg/utils"
)

type UpOptions struct {
	DockerCLI                *docker.CLI
	WorkspaceFolder          string
	ContainerWorkspaceFolder string
	ContainerName            string
	BaseImage                string
	OnCreateCommand          interface{}
	PostCreateCommand        interface{}
	DockerComposeFile        interface{}
	Service                  string
	ConfigPath               string
	Mounts                   []string
	WorkspaceMount           string
	RemoteUser               string
	Features                 map[string]interface{}
	ContainerEnv             map[string]string
}

// CleanProjectName cleans a string to be compatible with Docker Compose project names
func CleanProjectName(name string) string {
	var sb strings.Builder
	for _, r := range strings.ToLower(name) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			sb.WriteRune(r)
		}
	}
	res := sb.String()
	if res == "" {
		res = "devcontainer"
	}
	return res
}

// ParseStringOrSlice converts a string or []interface{} or []string to a []string
func ParseStringOrSlice(val interface{}) []string {
	if val == nil {
		return nil
	}
	if s, ok := val.(string); ok {
		return []string{s}
	}
	if slice, ok := val.([]interface{}); ok {
		var res []string
		for _, v := range slice {
			if s, ok := v.(string); ok {
				res = append(res, s)
			}
		}
		return res
	}
	if slice, ok := val.([]string); ok {
		return slice
	}
	return nil
}

// ResolveComposeFiles resolves dockerComposeFile paths relative to devcontainer.json directory
func ResolveComposeFiles(dockerComposeFile interface{}, configPath string) []string {
	configDir := filepath.Dir(configPath)
	var resolvedComposeFiles []string
	for _, f := range ParseStringOrSlice(dockerComposeFile) {
		if filepath.IsAbs(f) {
			resolvedComposeFiles = append(resolvedComposeFiles, f)
		} else {
			resolvedComposeFiles = append(resolvedComposeFiles, filepath.Join(configDir, f))
		}
	}
	return resolvedComposeFiles
}

func parseCommand(cmd interface{}) ([]string, bool) {
	if cmd == nil {
		return nil, false
	}
	if s, ok := cmd.(string); ok {
		if s == "" {
			return nil, false
		}
		return []string{"sh", "-c", s}, true
	}
	if slice, ok := cmd.([]string); ok {
		if len(slice) == 0 {
			return nil, false
		}
		return slice, true
	}
	if slice, ok := cmd.([]interface{}); ok {
		if len(slice) == 0 {
			return nil, false
		}
		var strSlice []string
		for _, v := range slice {
			strSlice = append(strSlice, fmt.Sprintf("%v", v))
		}
		return strSlice, true
	}
	return nil, false
}

func substituteEnvVar(value string, currentEnv map[string]string) string {
	for k, v := range currentEnv {
		value = strings.ReplaceAll(value, "${"+k+"}", v)
		value = strings.ReplaceAll(value, "${containerEnv:"+k+"}", v)
	}
	value = strings.ReplaceAll(value, "${PATH}", "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin")
	value = strings.ReplaceAll(value, "${containerEnv:PATH}", "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin")
	return value
}

type DevContainerFeatureMetadata struct {
	ID            string            `json:"id"`
	DependsOn     []string          `json:"dependsOn,omitempty"`
	InstallsAfter []string          `json:"installsAfter,omitempty"`
	ContainerEnv  map[string]string `json:"containerEnv,omitempty"`
}

func buildFeaturesImage(opts UpOptions, baseImage string) (string, map[string]string, error) {
	if len(opts.Features) == 0 {
		return baseImage, nil, nil
	}

	fmt.Println("Resolving and downloading features...")

	tmpDir, err := os.MkdirTemp("", "dc-features-build-*")
	if err != nil {
		return "", nil, fmt.Errorf("failed to create temp build directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	featuresDir := filepath.Join(tmpDir, "features")
	if err := os.MkdirAll(featuresDir, 0755); err != nil {
		return "", nil, fmt.Errorf("failed to create features directory: %w", err)
	}

	client := features.NewOCIClient(nil)

	// Keep track of resolved features for sorting and building
	var feats []features.Feature
	featureOptionsMap := make(map[string]map[string]interface{})
	featureRefsMap := make(map[string]features.FeatureRef)
	featureEnvMap := make(map[string]map[string]string)

	for refStr, val := range opts.Features {
		ref, err := features.ParseFeatureRef(refStr)
		if err != nil {
			return "", nil, fmt.Errorf("failed to parse feature ref %q: %w", refStr, err)
		}

		fullID := ref.Registry + "/" + ref.Namespace + "/" + ref.ID + ":" + ref.Version
		safeID := docker.GetSafeID(fullID)
		featDestDir := filepath.Join(featuresDir, safeID)
		if err := os.MkdirAll(featDestDir, 0755); err != nil {
			return "", nil, fmt.Errorf("failed to create directory for feature %s: %w", ref.ID, err)
		}

		fmt.Printf("Fetching feature manifest for %s...\n", refStr)
		manifest, err := client.FetchManifest(ref)
		if err != nil {
			return "", nil, fmt.Errorf("failed to fetch manifest for feature %s: %w", refStr, err)
		}

		if len(manifest.Layers) == 0 {
			return "", nil, fmt.Errorf("no layers found in manifest for feature %s", refStr)
		}

		tarGzPath := filepath.Join(tmpDir, safeID+".tar.gz")
		f, err := os.Create(tarGzPath)
		if err != nil {
			return "", nil, fmt.Errorf("failed to create tarball path for feature %s: %w", ref.ID, err)
		}

		fmt.Printf("Downloading feature blob for %s...\n", refStr)
		err = client.DownloadBlob(ref, manifest.Layers[0].Digest, f)
		f.Close()
		if err != nil {
			return "", nil, fmt.Errorf("failed to download blob for feature %s: %w", ref.ID, err)
		}

		fRead, err := os.Open(tarGzPath)
		if err != nil {
			return "", nil, fmt.Errorf("failed to open downloaded blob for feature %s: %w", ref.ID, err)
		}
		err = features.ExtractTarGz(fRead, featDestDir)
		fRead.Close()
		if err != nil {
			return "", nil, fmt.Errorf("failed to extract feature %s: %w", ref.ID, err)
		}

		// Read devcontainer-feature.json metadata
		var dependsOn []string
		var installsAfter []string
		var containerEnv map[string]string
		metaPath := filepath.Join(featDestDir, "devcontainer-feature.json")
		if data, err := os.ReadFile(metaPath); err == nil {
			var meta DevContainerFeatureMetadata
			if err := json.Unmarshal(data, &meta); err == nil {
				dependsOn = meta.DependsOn
				installsAfter = meta.InstallsAfter
				containerEnv = meta.ContainerEnv
			}
		}

		feats = append(feats, features.Feature{
			ID:            ref.ID,
			DependsOn:     dependsOn,
			InstallsAfter: installsAfter,
		})

		// Convert options safely
		optsMap := make(map[string]interface{})
		if m, ok := val.(map[string]interface{}); ok {
			optsMap = m
		}
		featureOptionsMap[ref.ID] = optsMap
		featureRefsMap[ref.ID] = ref
		featureEnvMap[ref.ID] = containerEnv
	}

	// Sort features based on dependencies
	sortedFeats, err := features.SortFeatures(feats)
	if err != nil {
		return "", nil, fmt.Errorf("failed to sort features: %w", err)
	}

	// Construct Docker BuildOptions Features list in sorted order
	var installOpts []docker.FeatureInstallOptions
	for _, f := range sortedFeats {
		ref := featureRefsMap[f.ID]
		fullID := ref.Registry + "/" + ref.Namespace + "/" + ref.ID + ":" + ref.Version
		installOpts = append(installOpts, docker.FeatureInstallOptions{
			ID:          fullID,
			UnpackedDir: filepath.Join(featuresDir, docker.GetSafeID(fullID)),
			Options:     featureOptionsMap[f.ID],
		})
	}

	// Build sorted features env
	featureEnv := make(map[string]string)
	for _, f := range sortedFeats {
		env := featureEnvMap[f.ID]
		for k, v := range env {
			featureEnv[k] = substituteEnvVar(v, featureEnv)
		}
	}

	targetImage := opts.ContainerName + "-features"

	buildOpts := docker.BuildOptions{
		BaseImage:   baseImage,
		TargetImage: targetImage,
		Features:    installOpts,
	}

	dockerfilePath := filepath.Join(tmpDir, "Dockerfile")
	dockerfileContent, err := docker.GenerateDockerfile(buildOpts)
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate Dockerfile for features: %w", err)
	}

	if err := os.WriteFile(dockerfilePath, []byte(dockerfileContent), 0644); err != nil {
		return "", nil, fmt.Errorf("failed to write Dockerfile for features: %w", err)
	}

	fmt.Printf("Building features layered image %s...\n", targetImage)
	err = opts.DockerCLI.BuildImage(dockerfilePath, tmpDir, buildOpts)
	if err != nil {
		return "", nil, fmt.Errorf("failed to build features image: %w", err)
	}

	return targetImage, featureEnv, nil
}

func RunUp(opts UpOptions) error {
	var targetContainer string

	// Build environment variables map
	mergedEnv := make(map[string]string)

	if opts.DockerComposeFile != nil && opts.Service != "" {
		// Resolve compose files
		resolvedComposeFiles := ResolveComposeFiles(opts.DockerComposeFile, opts.ConfigPath)
		projectName := CleanProjectName(filepath.Base(opts.WorkspaceFolder))

		var layeredImage string
		if len(opts.Features) > 0 {
			// 1. Get the base image for the service from compose file config
			composeConfig, err := opts.DockerCLI.ComposeConfig(resolvedComposeFiles)
			if err != nil {
				return fmt.Errorf("failed to get compose config: %w", err)
			}

			buildInfo, err := docker.ParseComposeConfig(composeConfig, opts.Service)
			if err != nil {
				return fmt.Errorf("failed to parse compose config: %w", err)
			}

			baseImage := buildInfo.Image
			if baseImage == "" {
				baseImage = "ubuntu:latest" // fallback if not explicitly defined
			}

			// 2. Build features layered image
			var featureEnv map[string]string
			layeredImage, featureEnv, err = buildFeaturesImage(opts, baseImage)
			if err != nil {
				return err
			}

			for k, v := range featureEnv {
				mergedEnv[k] = v
			}
		}

		// Merge config ContainerEnv
		for k, v := range opts.ContainerEnv {
			mergedEnv[k] = substituteEnvVar(v, mergedEnv)
		}

		if len(opts.Features) > 0 {
			// 3. Generate compose override file for the layered image
			overrideOpts := docker.ComposeOverrideOptions{
				Service: opts.Service,
				Image:   layeredImage,
				Env:     mergedEnv,
			}
			overrideYAML, err := docker.GenerateComposeOverride(overrideOpts)
			if err != nil {
				return fmt.Errorf("failed to generate compose override: %w", err)
			}

			tmpOverrideFile, err := os.CreateTemp("", "dc-compose-override-*.yml")
			if err != nil {
				return fmt.Errorf("failed to create temp override file: %w", err)
			}
			defer os.Remove(tmpOverrideFile.Name())

			if _, err := tmpOverrideFile.WriteString(overrideYAML); err != nil {
				tmpOverrideFile.Close()
				return fmt.Errorf("failed to write compose override file: %w", err)
			}
			tmpOverrideFile.Close()

			resolvedComposeFiles = append(resolvedComposeFiles, tmpOverrideFile.Name())
		}

		// Start compose stack
		fmt.Printf("Starting compose stack with project %s...\n", projectName)
		err := opts.DockerCLI.ComposeUp(resolvedComposeFiles, projectName)
		if err != nil {
			return fmt.Errorf("failed to start compose stack: %w", err)
		}

		// Get target service container
		cID, err := opts.DockerCLI.GetComposeServiceContainer(resolvedComposeFiles, projectName, opts.Service)
		if err != nil {
			return fmt.Errorf("failed to get container for service %s: %w", opts.Service, err)
		}
		if cID == "" {
			return fmt.Errorf("service %s has no running container", opts.Service)
		}
		targetContainer = cID
	} else {
		// 1. Inspect container
		_, err := opts.DockerCLI.InspectContainer(opts.ContainerName)
		containerExists := err == nil

		if !containerExists {
			baseImage := opts.BaseImage
			if len(opts.Features) > 0 {
				// 2. Inspect base image (and pull if not present)
				_, err = opts.DockerCLI.InspectImage(baseImage)
				if err != nil {
					fmt.Printf("Base image %s not found locally. Pulling...\n", baseImage)
					if pullErr := opts.DockerCLI.PullImage(baseImage); pullErr != nil {
						return fmt.Errorf("failed to pull base image %s: %w (inspect error: %v)", baseImage, pullErr, err)
					}
				}

				// Build features image
				var featureEnv map[string]string
				var layeredImage string
				layeredImage, featureEnv, err = buildFeaturesImage(opts, baseImage)
				if err != nil {
					return err
				}
				baseImage = layeredImage
				for k, v := range featureEnv {
					mergedEnv[k] = v
				}
			} else {
				// 2. Inspect base image (and pull if not present)
				_, err = opts.DockerCLI.InspectImage(baseImage)
				if err != nil {
					fmt.Printf("Base image %s not found locally. Pulling...\n", baseImage)
					if pullErr := opts.DockerCLI.PullImage(baseImage); pullErr != nil {
						return fmt.Errorf("failed to pull base image %s: %w (inspect error: %v)", baseImage, pullErr, err)
					}
				}
			}

			// Merge config ContainerEnv
			for k, v := range opts.ContainerEnv {
				mergedEnv[k] = substituteEnvVar(v, mergedEnv)
			}

			// 3. Configure RunOptions with mounts and env
			runOpts := docker.RunOptions{
				Image: baseImage,
				Name:  opts.ContainerName,
				Env:   make(map[string]string),
			}

			for k, v := range mergedEnv {
				runOpts.Env[k] = v
			}

			// Add workspace mount
			if opts.WorkspaceMount != "" {
				runOpts.Mounts = append(runOpts.Mounts, opts.WorkspaceMount)
			}

			// Add other custom mounts
			runOpts.Mounts = append(runOpts.Mounts, opts.Mounts...)

			// Expose SSH Agent socket if present
			if sshAuthSock := os.Getenv("SSH_AUTH_SOCK"); sshAuthSock != "" {
				translatedSSH := utils.TranslateWSLPath(sshAuthSock)
				runOpts.Mounts = append(runOpts.Mounts, fmt.Sprintf("type=bind,source=%s,target=/tmp/vscode-ssh-auth.sock", translatedSSH))
				runOpts.Env["SSH_AUTH_SOCK"] = "/tmp/vscode-ssh-auth.sock"
			}

			// Expose GPG Agent socket if present
			if gpgSock := gpgAgentSocketDetector(); gpgSock != "" {
				if _, err := os.Stat(gpgSock); err == nil {
					translatedGPG := utils.TranslateWSLPath(gpgSock)
					runOpts.Mounts = append(runOpts.Mounts, fmt.Sprintf("type=bind,source=%s,target=/tmp/gpg-agent.sock", translatedGPG))
					runOpts.Env["GPG_AGENT_SOCK"] = "/tmp/gpg-agent.sock"
				}
			}

			// Run container with standard keep-alive command
			runOpts.Cmd = []string{"sh", "-c", "echo Container started; trap \"exit 0\" 15; while sleep 1 & wait $!; do :; done"}
			_, err = opts.DockerCLI.RunContainer(runOpts)
			if err != nil {
				return fmt.Errorf("failed to start container %s: %w", opts.ContainerName, err)
			}
		}
		targetContainer = opts.ContainerName
	}

	// 4. Exec OnCreateCommand
	if cmdArgs, ok := parseCommand(opts.OnCreateCommand); ok {
		_, err := opts.DockerCLI.ExecCommandWithUser(targetContainer, "", opts.ContainerWorkspaceFolder, cmdArgs)
		if err != nil {
			return fmt.Errorf("failed to execute onCreateCommand: %w", err)
		}
	}

	// 5. Exec PostCreateCommand
	if cmdArgs, ok := parseCommand(opts.PostCreateCommand); ok {
		_, err := opts.DockerCLI.ExecCommandWithUser(targetContainer, opts.RemoteUser, opts.ContainerWorkspaceFolder, cmdArgs)
		if err != nil {
			return fmt.Errorf("failed to execute postCreateCommand: %w", err)
		}
	}

	return nil
}

// ResolveContainerName finds the actual container ID if Compose is used, otherwise returns defaultName
func ResolveContainerName(wsFolder string, cfgPath string, defaultName string, dCli *docker.CLI) (string, error) {
	if cfgPath == "" {
		p1 := filepath.Join(wsFolder, ".devcontainer", "devcontainer.json")
		p2 := filepath.Join(wsFolder, ".devcontainer.json")
		if _, err := os.Stat(p1); err == nil {
			cfgPath = p1
		} else if _, err := os.Stat(p2); err == nil {
			cfgPath = p2
		}
	}

	if cfgPath != "" {
		data, err := os.ReadFile(cfgPath)
		if err == nil {
			parsed, err := config.Parse(string(data))
			if err == nil && parsed.DockerComposeFile != nil && parsed.Service != "" {
				resolvedComposeFiles := ResolveComposeFiles(parsed.DockerComposeFile, cfgPath)
				projectName := CleanProjectName(filepath.Base(wsFolder))
				cID, err := dCli.GetComposeServiceContainer(resolvedComposeFiles, projectName, parsed.Service)
				if err == nil && cID != "" {
					return cID, nil
				}
			}
		}
	}
	return defaultName, nil
}
