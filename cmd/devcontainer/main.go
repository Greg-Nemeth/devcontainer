package main

import (
	"fmt"
	"os"
	"regexp"

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
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Running up...")
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
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Running exec...")
		},
	}

	readConfigCmd := &cobra.Command{
		Use:   "read-configuration",
		Short: "Read configuration",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Running read-configuration...")
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
