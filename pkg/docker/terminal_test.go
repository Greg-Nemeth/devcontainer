package docker

import (
	"os/exec"
	"testing"
)

func TestIsStdinTerminalCheck(t *testing.T) {
	isTerm := IsStdinTerminal()
	t.Logf("Is stdin terminal: %v", isTerm)
}

func TestRunInteractiveSubprocessSafety(t *testing.T) {
	// Verify that running a standard command works correctly under testing (non-TTY)
	cmd := exec.Command("echo", "terminal-test")
	err := RunInteractiveSubprocess(cmd)
	if err != nil {
		t.Fatalf("RunInteractiveSubprocess failed under non-TTY: %v", err)
	}
}
