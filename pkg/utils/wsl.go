package utils

import (
	"os"
	"os/exec"
	"strings"
)

var wslpathRunner = func(path string) (string, error) {
	cmd := exec.Command("wslpath", "-w", path)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

var isWSLOverride *bool

func IsWSL() bool {
	if isWSLOverride != nil {
		return *isWSLOverride
	}
	if os.Getenv("WSL_DISTRO_NAME") != "" {
		return true
	}
	data, err := os.ReadFile("/proc/sys/kernel/osrelease")
	if err == nil {
		content := strings.ToLower(string(data))
		if strings.Contains(content, "microsoft") || strings.Contains(content, "wsl") {
			return true
		}
	}
	return false
}

func TranslateWSLPath(path string) string {
	if !IsWSL() {
		return path
	}

	// Translate only paths mounted on WSL under /mnt/ (e.g. Windows drives)
	if strings.HasPrefix(path, "/mnt/") {
		winPath, err := wslpathRunner(path)
		if err == nil && winPath != "" {
			return winPath
		}
	}

	return path
}
