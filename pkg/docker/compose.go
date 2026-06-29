package docker

import (
	"strings"
)

func (c *CLI) ComposeDown(composeFiles []string, projectName string) error {
	var args []string
	for _, f := range composeFiles {
		args = append(args, "-f", f)
	}
	if projectName != "" {
		args = append(args, "-p", projectName)
	}
	args = append(args, "down")

	_, err := c.runner(c.ComposePath, args...)
	return err
}

func (c *CLI) GetComposeServiceContainer(composeFiles []string, projectName string, service string) (string, error) {
	var args []string
	for _, f := range composeFiles {
		args = append(args, "-f", f)
	}
	if projectName != "" {
		args = append(args, "-p", projectName)
	}
	args = append(args, "ps", "-q", service)

	output, err := c.runner(c.ComposePath, args...)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}
