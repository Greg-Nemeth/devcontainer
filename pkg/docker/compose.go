package docker

import (
	"encoding/json"
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

	// 1. First try: call `ps -q <service>` (standard docker-compose behaviour)
	argsWithService := append(args, "ps", "-q", service)
	output, err := c.runner(c.ComposePath, argsWithService...)
	if err == nil {
		id := strings.TrimSpace(string(output))
		if id != "" {
			return id, nil
		}
	}

	// 2. Fallback (for podman-compose compatibility where service name cannot be a positional arg to ps):
	// Call `ps -q` to get all containers, then inspect labels to find matching service
	argsWithoutService := append(args, "ps", "-q")
	output, err = c.runner(c.ComposePath, argsWithoutService...)
	if err != nil {
		return "", err
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, rawLine := range lines {
		cID := strings.TrimSpace(rawLine)
		if cID == "" {
			continue
		}

		inspectBytes, err := c.InspectContainer(cID)
		if err != nil {
			continue
		}

		var inspectList []struct {
			Config struct {
				Labels map[string]string `json:"Labels"`
			} `json:"Config"`
		}
		if err := json.Unmarshal(inspectBytes, &inspectList); err == nil && len(inspectList) > 0 {
			labels := inspectList[0].Config.Labels
			if labels["com.docker.compose.service"] == service || labels["io.podman.compose.service"] == service {
				return cID, nil
			}
		}
	}

	return "", nil
}
