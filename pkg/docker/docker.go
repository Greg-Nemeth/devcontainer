package docker

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
)

type RunnerFunc func(path string, args ...string) ([]byte, error)
type InteractiveRunnerFunc func(path string, args ...string) error

type CLI struct {
	CLIPath           string
	ComposePath       string
	runner            RunnerFunc
	interactiveRunner InteractiveRunnerFunc
}

type RunOptions struct {
	Image  string
	Name   string
	Mounts []string
	Env    map[string]string
	Cmd    []string
}

func DefaultRunner(path string, args ...string) ([]byte, error) {
	cmd := exec.Command(path, args...)

	if len(args) > 0 && (args[0] == "build" || args[0] == "pull") {
		var buf bytes.Buffer
		cmd.Stdout = io.MultiWriter(os.Stdout, &buf)
		cmd.Stderr = io.MultiWriter(os.Stderr, &buf)
		err := cmd.Run()
		return buf.Bytes(), err
	}

	return cmd.CombinedOutput()
}

func defaultInteractiveRunner(path string, args ...string) error {
	cmd := exec.Command(path, args...)
	return RunInteractiveSubprocess(cmd)
}

func NewCLI(cliPath, composePath string, runner RunnerFunc) *CLI {
	if runner == nil {
		runner = DefaultRunner
	}
	if cliPath == "" {
		cliPath = "docker"
	}
	if composePath == "" {
		composePath = "docker-compose"
	}
	return &CLI{
		CLIPath:           cliPath,
		ComposePath:       composePath,
		runner:            runner,
		interactiveRunner: defaultInteractiveRunner,
	}
}

func (c *CLI) SetInteractiveRunner(r InteractiveRunnerFunc) {
	c.interactiveRunner = r
}

func (c *CLI) ExecInteractive(containerID string, user string, workDir string, cmd []string) error {
	var args []string
	if IsStdinTerminal() {
		args = append(args, "exec", "-it")
	} else {
		args = append(args, "exec", "-i")
	}
	if user != "" {
		args = append(args, "-u", user)
	}
	if workDir != "" {
		args = append(args, "-w", workDir)
	}
	args = append(args, containerID)
	args = append(args, cmd...)

	return c.interactiveRunner(c.CLIPath, args...)
}

func (c *CLI) InspectContainer(id string) ([]byte, error) {
	return c.runner(c.CLIPath, "inspect", id)
}

func (c *CLI) InspectImage(name string) ([]byte, error) {
	return c.runner(c.CLIPath, "inspect", name)
}

func (c *CLI) RunContainer(opts RunOptions) ([]byte, error) {
	args := []string{"run", "-d"}
	if opts.Name != "" {
		args = append(args, "--name", opts.Name)
	}
	for _, m := range opts.Mounts {
		args = append(args, "--mount", m)
	}

	// Deterministic sorting of env keys for tests
	var envKeys []string
	for k := range opts.Env {
		envKeys = append(envKeys, k)
	}
	sort.Strings(envKeys)

	for _, k := range envKeys {
		args = append(args, "-e", fmt.Sprintf("%s=%s", k, opts.Env[k]))
	}

	args = append(args, opts.Image)
	args = append(args, opts.Cmd...)

	return c.runner(c.CLIPath, args...)
}

func (c *CLI) ComposeUp(composeFiles []string, projectName string) error {
	var args []string
	for _, f := range composeFiles {
		args = append(args, "-f", f)
	}
	if projectName != "" {
		args = append(args, "-p", projectName)
	}
	args = append(args, "up", "-d")

	_, err := c.runner(c.ComposePath, args...)
	return err
}

func (c *CLI) ExecCommand(id string, cmd []string) ([]byte, error) {
	args := append([]string{"exec", id}, cmd...)
	return c.runner(c.CLIPath, args...)
}

func (c *CLI) ExecCommandWithUser(id string, user string, workDir string, cmd []string) ([]byte, error) {
	var args []string
	args = append(args, "exec")
	if user != "" {
		args = append(args, "-u", user)
	}
	if workDir != "" {
		args = append(args, "-w", workDir)
	}
	args = append(args, id)
	args = append(args, cmd...)
	return c.runner(c.CLIPath, args...)
}

func (c *CLI) PullImage(imageName string) error {
	_, err := c.runner(c.CLIPath, "pull", imageName)
	return err
}
