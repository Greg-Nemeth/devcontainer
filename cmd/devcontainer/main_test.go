package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestCLICommandsRegistered(t *testing.T) {
	rootCmd := newRootCommand()
	
	subcommands := []string{"up", "build", "exec", "read-configuration"}
	for _, sub := range subcommands {
		found := false
		for _, cmd := range rootCmd.Commands() {
			if cmd.Name() == sub {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected subcommand %q to be registered", sub)
		}
	}
}

func TestCLIHelpMessage(t *testing.T) {
	rootCmd := newRootCommand()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"--help"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Help command failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "devcontainer") {
		t.Errorf("Expected help output to contain 'devcontainer', got %q", output)
	}
}

func TestCLIIdLabelValidation(t *testing.T) {
	rootCmd := newRootCommand()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	
	// Test invalid id-label format (missing equals sign)
	rootCmd.SetArgs([]string{"up", "--id-label", "invalidlabelformat"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("Expected error on invalid id-label format, but none was returned")
	}

	if !strings.Contains(err.Error(), "id-label must match <name>=<value>") {
		t.Errorf("Expected id-label validation error message, got %q", err.Error())
	}
}
