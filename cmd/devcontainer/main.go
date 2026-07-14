package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/devcontainers/dc/pkg/cli"
	"github.com/devcontainers/dc/pkg/config"
	"github.com/devcontainers/dc/pkg/docker"
	"github.com/devcontainers/dc/pkg/features"
	"github.com/devcontainers/dc/pkg/templates"
	"github.com/spf13/cobra"
	"strings"
)

var idLabelRegex = regexp.MustCompile(`^[^=]+=[^=]+$`)

type devcontainerOptions struct {
	dockerPath        string
	dockerComposePath string
	workspaceFolder   string
	configPath        string
	overrideConfig    string
	logLevel          string
	logFormat         string
	idLabels          []string
}

func newRootCommand() *cobra.Command {
	opts := &devcontainerOptions{}

	rootCmd := &cobra.Command{
		Use:   "devcontainer",
		Short: "Dev Containers CLI in Go",
		Long:  "An independent, native Go translation of the official devcontainers/cli project.",
	}

	// Persistent flags (available to all subcommands)
	rootCmd.PersistentFlags().StringVar(&opts.dockerPath, "docker-path", "", "Docker CLI path")
	rootCmd.PersistentFlags().StringVar(&opts.dockerComposePath, "docker-compose-path", "", "Docker Compose CLI path")
	rootCmd.PersistentFlags().StringVar(&opts.workspaceFolder, "workspace-folder", "", "Workspace folder path")
	rootCmd.PersistentFlags().StringVar(&opts.configPath, "config", "", "devcontainer.json path")
	rootCmd.PersistentFlags().StringVar(&opts.overrideConfig, "override-config", "", "devcontainer.json path to override")
	rootCmd.PersistentFlags().StringVar(&opts.logLevel, "log-level", "info", "Log level (info, debug, trace)")
	rootCmd.PersistentFlags().StringVar(&opts.logFormat, "log-format", "text", "Log format (text, json)")
	rootCmd.PersistentFlags().StringSliceVar(&opts.idLabels, "id-label", nil, "Id label(s) of the format name=value")

	upCmd := &cobra.Command{
		Use:   "up",
		Short: "Create and run dev container",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// Validate id-labels format
			for _, label := range opts.idLabels {
				if !idLabelRegex.MatchString(label) {
					return fmt.Errorf("id-label must match <name>=<value>")
				}
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			wsFolder := opts.workspaceFolder
			if wsFolder == "" {
				var err error
				wsFolder, err = os.Getwd()
				if err != nil {
					return err
				}
			}
			wsFolder, _ = filepath.Abs(wsFolder)

			// 1. Find and parse devcontainer config
			cfgPath := opts.configPath
			if cfgPath == "" {
				p1 := filepath.Join(wsFolder, ".devcontainer", "devcontainer.json")
				p2 := filepath.Join(wsFolder, ".devcontainer.json")
				if _, err := os.Stat(p1); err == nil {
					cfgPath = p1
				} else if _, err := os.Stat(p2); err == nil {
					cfgPath = p2
				}
			}

			var baseImage string
			var onCreateCmd, postCreateCmd interface{}
			var dockerComposeFile interface{}
			var service string
			var mounts []string
			var workspaceMount string
			var remoteUser string

			if cfgPath != "" {
				data, err := os.ReadFile(cfgPath)
				if err != nil {
					return err
				}
				parsed, err := config.Parse(string(data))
				if err != nil {
					return err
				}

				parsed.SubstituteVariables(wsFolder, cfgPath)

				baseImage = parsed.Image
				onCreateCmd = parsed.OnCreateCommand
				postCreateCmd = parsed.PostCreateCommand
				dockerComposeFile = parsed.DockerComposeFile
				service = parsed.Service
				remoteUser = parsed.RemoteUser

				for _, m := range parsed.Mounts {
					if s, ok := m.(string); ok {
						mounts = append(mounts, s)
					}
				}

				workspaceMount = parsed.WorkspaceMount
				if workspaceMount == "" {
					wsFolderTarget := parsed.WorkspaceFolder
					if wsFolderTarget == "" {
						wsFolderTarget = "/workspace"
					}
					workspaceMount = fmt.Sprintf("type=bind,source=%s,target=%s", wsFolder, wsFolderTarget)
				}
			} else {
				workspaceMount = fmt.Sprintf("type=bind,source=%s,target=/workspace", wsFolder)
			}

			if baseImage == "" {
				baseImage = "ubuntu:latest" // Default fallback
			}

			// 2. Initialize Docker CLI client
			dockerPath := opts.dockerPath
			if dockerPath == "" {
				dockerPath = "docker" // Default
			}
			dCli := docker.NewCLI(dockerPath, opts.dockerComposePath, nil)

			// 3. Trigger up workflow
			cName := fmt.Sprintf("devcontainer-%s", filepath.Base(wsFolder))
			containerWorkspaceFolder := "/workspace"
			if cfgPath != "" {
				// We already parsed config, let's extract parsed.WorkspaceFolder
				data, err := os.ReadFile(cfgPath)
				if err == nil {
					parsed, err := config.Parse(string(data))
					if err == nil {
						parsed.SubstituteVariables(wsFolder, cfgPath)
						if parsed.WorkspaceFolder != "" {
							containerWorkspaceFolder = parsed.WorkspaceFolder
						}
					}
				}
			}

			upOpts := cli.UpOptions{
				DockerCLI:                dCli,
				WorkspaceFolder:          wsFolder,
				ContainerWorkspaceFolder: containerWorkspaceFolder,
				ContainerName:            cName,
				BaseImage:                baseImage,
				OnCreateCommand:          onCreateCmd,
				PostCreateCommand:        postCreateCmd,
				DockerComposeFile:        dockerComposeFile,
				Service:                  service,
				ConfigPath:               cfgPath,
				Mounts:                   mounts,
				WorkspaceMount:           workspaceMount,
				RemoteUser:               remoteUser,
			}

			fmt.Printf("Orchestrating up workflow for container: %s...\n", cName)
			return cli.RunUp(upOpts)
		},
	}

	buildCmd := &cobra.Command{
		Use:   "build [path]",
		Short: "Build a dev container image",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Running build...")
		},
	}

	execCmd := &cobra.Command{
		Use:   "exec [cmd] [args...]",
		Short: "Execute a command on a running dev container",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("command and arguments must be supplied")
			}
			wsFolder := opts.workspaceFolder
			if wsFolder == "" {
				var err error
				wsFolder, err = os.Getwd()
				if err != nil {
					return err
				}
			}
			wsFolder, _ = filepath.Abs(wsFolder)

			dockerPath := opts.dockerPath
			if dockerPath == "" {
				dockerPath = "docker"
			}
			dCli := docker.NewCLI(dockerPath, opts.dockerComposePath, nil)

			cName := fmt.Sprintf("devcontainer-%s", filepath.Base(wsFolder))
			targetContainer, err := cli.ResolveContainerName(wsFolder, opts.configPath, cName, dCli)
			if err != nil {
				return err
			}

			// Parse devcontainer.json to get remoteUser
			cfgPath := opts.configPath
			if cfgPath == "" {
				p1 := filepath.Join(wsFolder, ".devcontainer", "devcontainer.json")
				p2 := filepath.Join(wsFolder, ".devcontainer.json")
				if _, err := os.Stat(p1); err == nil {
					cfgPath = p1
				} else if _, err := os.Stat(p2); err == nil {
					cfgPath = p2
				}
			}

			var remoteUser string
			var containerWorkspaceFolder string
			if cfgPath != "" {
				data, err := os.ReadFile(cfgPath)
				if err == nil {
					parsed, err := config.Parse(string(data))
					if err == nil {
						parsed.SubstituteVariables(wsFolder, cfgPath)
						remoteUser = parsed.RemoteUser
						containerWorkspaceFolder = parsed.WorkspaceFolder
					}
				}
			}
			if containerWorkspaceFolder == "" {
				containerWorkspaceFolder = "/workspace"
			}

			execOpts := cli.ExecOptions{
				DockerCLI:                dCli,
				ContainerName:            targetContainer,
				User:                     remoteUser,
				ContainerWorkspaceFolder: containerWorkspaceFolder,
				Command:                  args,
			}

			fmt.Printf("Executing command inside container %s: %v...\n", targetContainer, args)
			return cli.RunExec(execOpts)
		},
	}

	readConfigCmd := &cobra.Command{
		Use:   "read-configuration",
		Short: "Read configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			wsFolder := opts.workspaceFolder
			if wsFolder == "" {
				var err error
				wsFolder, err = os.Getwd()
				if err != nil {
					return err
				}
			}
			wsFolder, _ = filepath.Abs(wsFolder)

			cfgPath := opts.configPath
			if cfgPath == "" {
				p1 := filepath.Join(wsFolder, ".devcontainer", "devcontainer.json")
				p2 := filepath.Join(wsFolder, ".devcontainer.json")
				if _, err := os.Stat(p1); err == nil {
					cfgPath = p1
				} else if _, err := os.Stat(p2); err == nil {
					cfgPath = p2
				}
			}

			if cfgPath == "" {
				return fmt.Errorf("no devcontainer.json configuration file found in workspace")
			}

			data, err := os.ReadFile(cfgPath)
			if err != nil {
				return err
			}

			parsed, err := config.Parse(string(data))
			if err != nil {
				return err
			}

			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(parsed)
		},
	}

	var serverType string
	injectServerCmd := &cobra.Command{
		Use:   "inject-server",
		Short: "Inject and run a headless IDE server inside the container",
		RunE: func(cmd *cobra.Command, args []string) error {
			wsFolder := opts.workspaceFolder
			if wsFolder == "" {
				var err error
				wsFolder, err = os.Getwd()
				if err != nil {
					return err
				}
			}
			wsFolder, _ = filepath.Abs(wsFolder)

			dockerPath := opts.dockerPath
			if dockerPath == "" {
				dockerPath = "docker"
			}
			dCli := docker.NewCLI(dockerPath, opts.dockerComposePath, nil)

			cName := fmt.Sprintf("devcontainer-%s", filepath.Base(wsFolder))
			targetContainer, err := cli.ResolveContainerName(wsFolder, opts.configPath, cName, dCli)
			if err != nil {
				return err
			}

			if serverType == "" {
				serverType = "openvscode"
			}

			fmt.Printf("Injecting %s headless server inside container %s...\n", serverType, targetContainer)
			return cli.InjectHeadlessServer(dCli, targetContainer, serverType)
		},
	}
	injectServerCmd.Flags().StringVar(&serverType, "type", "openvscode", "IDE Server type (openvscode, jetbrains)")

	var templateID string
	var templateOptions []string

	templatesCmd := &cobra.Command{
		Use:   "templates",
		Short: "Manage dev container templates",
	}

	templatesApplyCmd := &cobra.Command{
		Use:   "apply",
		Short: "Apply a template to a workspace folder",
		RunE: func(cmd *cobra.Command, args []string) error {
			wsFolder := opts.workspaceFolder
			if wsFolder == "" {
				var err error
				wsFolder, err = os.Getwd()
				if err != nil {
					return err
				}
			}
			wsFolder, _ = filepath.Abs(wsFolder)

			if templateID == "" {
				return fmt.Errorf("--template is required")
			}

			// Parse options into map
			optMap := make(map[string]string)
			for _, opt := range templateOptions {
				parts := strings.SplitN(opt, "=", 2)
				if len(parts) == 2 {
					optMap[parts[0]] = parts[1]
				}
			}

			// Parse OCI template ref
			ref, err := templates.ParseTemplateRef(templateID)
			if err != nil {
				return err
			}

			fmt.Printf("Fetching template %s...\n", templateID)

			tmpDir, err := os.MkdirTemp("", "dc-template-download-*")
			if err != nil {
				return err
			}
			defer os.RemoveAll(tmpDir)

			client := features.NewOCIClient(nil)
			manifest, err := client.FetchManifest(features.FeatureRef{
				Registry:  ref.Registry,
				Namespace: ref.Namespace,
				ID:        ref.ID,
				Version:   ref.Version,
			})
			if err != nil {
				return fmt.Errorf("failed to fetch template manifest: %w", err)
			}

			if len(manifest.Layers) == 0 {
				return fmt.Errorf("no layers found in template manifest")
			}

			// Download first layer blob (tar.gz)
			tarGzPath := filepath.Join(tmpDir, "layer.tar.gz")
			f, err := os.Create(tarGzPath)
			if err != nil {
				return err
			}

			err = client.DownloadBlob(features.FeatureRef{
				Registry:  ref.Registry,
				Namespace: ref.Namespace,
				ID:        ref.ID,
				Version:   ref.Version,
			}, manifest.Layers[0].Digest, f)
			f.Close()
			if err != nil {
				return fmt.Errorf("failed to download template layer: %w", err)
			}

			// Extract tarball
			unpackedDir := filepath.Join(tmpDir, "unpacked")
			os.MkdirAll(unpackedDir, 0755)

			fRead, err := os.Open(tarGzPath)
			if err != nil {
				return err
			}
			err = features.ExtractTarGz(fRead, unpackedDir)
			fRead.Close()
			if err != nil {
				return fmt.Errorf("failed to extract template tarball: %w", err)
			}

			// Apply template
			fmt.Printf("Applying template to %s...\n", wsFolder)
			err = templates.ApplyTemplate(unpackedDir, wsFolder, optMap)
			if err != nil {
				return fmt.Errorf("failed to apply template: %w", err)
			}

			fmt.Println("Template applied successfully!")
			return nil
		},
	}

	templatesApplyCmd.Flags().StringVar(&templateID, "template", "", "Template ID (e.g. ghcr.io/devcontainers/templates/go:1)")
	templatesApplyCmd.Flags().StringSliceVar(&templateOptions, "option", nil, "Template options (e.g. goVersion=1.21)")

	templatesCmd.AddCommand(templatesApplyCmd)

	// Add subcommands
	rootCmd.AddCommand(upCmd)
	rootCmd.AddCommand(buildCmd)
	rootCmd.AddCommand(execCmd)
	rootCmd.AddCommand(readConfigCmd)
	rootCmd.AddCommand(injectServerCmd)
	rootCmd.AddCommand(templatesCmd)

	return rootCmd
}

func main() {
	rootCmd := newRootCommand()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
