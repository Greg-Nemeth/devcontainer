package docker

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"golang.org/x/term"
)

func IsStdinTerminal() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

func RunInteractiveSubprocess(cmd *exec.Cmd) error {
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	isTerm := IsStdinTerminal()
	if isTerm {
		oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
		if err == nil {
			defer func() {
				_ = term.Restore(int(os.Stdin.Fd()), oldState)
			}()
		}
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start subprocess: %w", err)
	}

	// Setup signal forwarding
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGWINCH)
	defer func() {
		signal.Stop(sigChan)
		close(sigChan)
	}()

	go func() {
		for sig := range sigChan {
			if cmd.Process != nil {
				_ = cmd.Process.Signal(sig)
			}
		}
	}()

	return cmd.Wait()
}
