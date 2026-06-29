package utils

import (
	"strings"
	"testing"
)

func TestIsWSLCheck(t *testing.T) {
	isW := IsWSL()
	t.Logf("Is WSL detected: %v", isW)
}

func TestTranslateWSLPath(t *testing.T) {
	trueVal := true
	isWSLOverride = &trueVal
	oldRunner := wslpathRunner
	wslpathRunner = func(path string) (string, error) {
		if strings.HasPrefix(path, "/mnt/c/") {
			return "C:\\" + strings.ReplaceAll(path[7:], "/", "\\"), nil
		}
		return "", nil
	}
	defer func() {
		wslpathRunner = oldRunner
		isWSLOverride = nil
	}()

	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "/mnt/c/Projects/demo",
			expected: `C:\Projects\demo`,
		},
		{
			input:    "/home/user/project",
			expected: "/home/user/project", // Returns as-is
		},
	}

	for _, tc := range tests {
		got := TranslateWSLPath(tc.input)
		if got != tc.expected {
			t.Errorf("TranslateWSLPath(%q) = %q; want %q", tc.input, got, tc.expected)
		}
	}
}
