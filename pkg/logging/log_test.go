package logging

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected LogLevel
	}{
		{"trace", LevelTrace},
		{"debug", LevelDebug},
		{"info", LevelInfo},
		{"warning", LevelWarning},
		{"error", LevelError},
		{"critical", LevelCritical},
		{"off", LevelOff},
		{"unknown", LevelInfo}, // Default fallback
	}

	for _, tc := range tests {
		got := ParseLogLevel(tc.input)
		if got != tc.expected {
			t.Errorf("ParseLogLevel(%q) = %v; want %v", tc.input, got, tc.expected)
		}
	}
}

func TestColorAndFormatting(t *testing.T) {
	// Test color wrapping
	str := "hello"
	colorCode := "38;2;143;99;79"
	colored := Color(colorCode, str)
	if !strings.Contains(colored, "hello") {
		t.Errorf("Expected colored string to contain %q, got %q", str, colored)
	}

	// Test stripping escape sequences
	withEscapes := "hello \x1B[1mworld\x1B[0m"
	stripped := StripEscapeSequences(withEscapes)
	if stripped != "hello world" {
		t.Errorf("StripEscapeSequences(%q) = %q; want %q", withEscapes, stripped, "hello world")
	}
}

func TestColorize(t *testing.T) {
	// Colorize should turn numbers to blue (or whatever color we define)
	text := "Version 1.25.11 has 42 items"
	colored := Colorize(text)
	if !strings.Contains(colored, "1.25.11") || !strings.Contains(colored, "42") {
		t.Errorf("Colorize lost content, got: %q", colored)
	}
}

type mockWriter struct {
	outputs []string
}

func (m *mockWriter) Write(p []byte) (n int, err error) {
	m.outputs = append(m.outputs, string(p))
	return len(p), nil
}

func TestPlainLogger(t *testing.T) {
	mock := &mockWriter{}
	logger := NewPlainLogger(mock, LevelDebug)

	logger.Write("test debug log", LevelDebug)
	logger.Write("test trace log", LevelTrace) // Should be skipped since level is LevelDebug

	if len(mock.outputs) != 1 {
		t.Fatalf("Expected 1 output, got %d", len(mock.outputs))
	}
	if !strings.Contains(mock.outputs[0], "test debug log") {
		t.Errorf("Expected output to contain 'test debug log', got %q", mock.outputs[0])
	}
}

func TestJSONLogger(t *testing.T) {
	mock := &mockWriter{}
	logger := NewJSONLogger(mock, LevelInfo)

	logger.Write("json test info", LevelInfo)

	if len(mock.outputs) != 1 {
		t.Fatalf("Expected 1 output, got %d", len(mock.outputs))
	}

	var event LogEvent
	if err := json.Unmarshal([]byte(mock.outputs[0]), &event); err != nil {
		t.Fatalf("Failed to unmarshal JSON log: %v", err)
	}

	if event.Type != "text" {
		t.Errorf("Expected event.Type to be 'text', got %q", event.Type)
	}
	if event.Text != "json test info\n" {
		t.Errorf("Expected event.Text to be 'json test info\n', got %q", event.Text)
	}
	if event.Level != LevelInfo {
		t.Errorf("Expected event.Level to be %v, got %v", LevelInfo, event.Level)
	}
}

func TestStopLog(t *testing.T) {
	mock := &mockWriter{}
	logger := NewPlainLogger(mock, LevelDebug)

	start := logger.Start("operation start", LevelDebug)
	time.Sleep(2 * time.Millisecond)
	logger.Stop("operation stop", start, LevelDebug)

	if len(mock.outputs) != 2 {
		t.Fatalf("Expected 2 outputs, got %d", len(mock.outputs))
	}
	if !strings.Contains(mock.outputs[0], "Start: operation start") {
		t.Errorf("Expected start log, got %q", mock.outputs[0])
	}
	if !strings.Contains(mock.outputs[1], "Stop") {
		t.Errorf("Expected stop log, got %q", mock.outputs[1])
	}
}
