package cli

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/devcontainers/dc/pkg/docker"
)

var gpgAgentSocketDetector = detectGPGAgentSocket

func detectGPGAgentSocket() string {
	cmd := exec.Command("gpgconf", "--list-dirs", "agent-socket")
	out, err := cmd.Output()
	if err == nil {
		path := strings.TrimSpace(string(out))
		if path != "" {
			return path
		}
	}
	return filepath.Join(os.Getenv("HOME"), ".gnupg", "S.gpg-agent")
}

func getBootstrapperScript(serverType string) (string, error) {
	switch serverType {
	case "openvscode":
		return `#!/bin/sh
set -e
mkdir -p /tmp/devcontainer-server
cd /tmp/devcontainer-server
URL="https://github.com/gitpod-io/openvscode-server/releases/download/openvscode-server-v1.85.1/openvscode-server-v1.85.1-linux-x64.tar.gz"
echo "Downloading OpenVSCode Server..."
if command -v curl >/dev/null; then
    curl -sSL -o server.tar.gz "$URL"
else
    wget -q -O server.tar.gz "$URL"
fi
echo "Extracting..."
tar -xzf server.tar.gz --strip-components=1
echo "Starting OpenVSCode Server on port 3000..."
./bin/openvscode-server --host 0.0.0.0 --port 3000 --without-connection-token >/tmp/server.log 2>&1 &
echo "Done"
`, nil
	case "jetbrains":
		return `#!/bin/sh
set -e
echo "Starting mock JetBrains installer..."
# Simply print installation start
`, nil
	default:
		return "", fmt.Errorf("unsupported server type: %s", serverType)
	}
}

func InjectHeadlessServer(dockerCLI *docker.CLI, containerName, serverType string) error {
	script, err := getBootstrapperScript(serverType)
	if err != nil {
		return err
	}

	// We execute "sh" inside the container, piping the bootstrapper script into stdin.
	args := []string{"exec", "-i", containerName, "sh"}
	cmd := exec.Command(dockerCLI.CLIPath, args...)
	cmd.Stdin = bytes.NewReader([]byte(script))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
