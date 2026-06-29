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
	"github.com/spf13/cobra"
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
			if cfgPath != "" {
				data, err := os.ReadFile(cfgPath)
				if err != nil {
					return err
				}
				parsed, err := config.Parse(string(data))
				if err != nil {
					return err
				}
				baseImage = parsed.Image
				onCreateCmd = parsed.OnCreateCommand
				postCreateCmd = parsed.PostCreateCommand
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
			upOpts := cli.UpOptions{
				DockerCLI:         dCli,
				WorkspaceFolder:   wsFolder,
				ContainerName:     cName,
				BaseImage:         baseImage,
				OnCreateCommand:   onCreateCmd,
				PostCreateCommand: postCreateCmd,
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
			execOpts := cli.ExecOptions{
				DockerCLI:     dCli,
				ContainerName: cName,
				Command:       args,
			}

			fmt.Printf("Executing command inside container %s: %v...\n", cName, args)
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

	// Add subcommands
	rootCmd.AddCommand(upCmd)
	rootCmd.AddCommand(buildCmd)
	rootCmd.AddCommand(execCmd)
	rootCmd.AddCommand(readConfigCmd)

	return rootCmd
}

func main() {
	rootCmd := newRootCommand()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
