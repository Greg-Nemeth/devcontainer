package logging

import (
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type LogLevel int

const (
	LevelTrace LogLevel = iota + 1
	LevelDebug
	LevelInfo
	LevelWarning
	LevelError
	LevelCritical
	LevelOff
)

func (l LogLevel) String() string {
	switch l {
	case LevelTrace:
		return "trace"
	case LevelDebug:
		return "debug"
	case LevelInfo:
		return "info"
	case LevelWarning:
		return "warning"
	case LevelError:
		return "error"
	case LevelCritical:
		return "critical"
	case LevelOff:
		return "off"
	default:
		return "info"
	}
}

func ParseLogLevel(text string) LogLevel {
	switch strings.ToLower(text) {
	case "trace":
		return LevelTrace
	case "debug":
		return LevelDebug
	case "info":
		return LevelInfo
	case "warning":
		return LevelWarning
	case "error":
		return LevelError
	case "critical":
		return LevelCritical
	case "off":
		return LevelOff
	default:
		return LevelInfo
	}
}

type LogEvent struct {
	Type           string   `json:"type"`
	Channel        string   `json:"channel,omitempty"`
	Level          LogLevel `json:"level"`
	Timestamp      int64    `json:"timestamp"`
	Text           string   `json:"text,omitempty"`
	StartTimestamp int64    `json:"startTimestamp,omitempty"`
	Name           string   `json:"name,omitempty"`
	Status         string   `json:"status,omitempty"`
	StepDetail     string   `json:"stepDetail,omitempty"`
}

type Log interface {
	Write(text string, level LogLevel)
	Raw(text string, level LogLevel)
	Start(text string, level LogLevel) int64
	Stop(text string, start int64, level LogLevel)
}

// Color codes
const (
	ColorRed       = "38;2;143;99;79"
	ColorGreen     = "38;2;99;143;79"
	ColorBlue      = "38;2;86;156;214"
	StopColor      = ColorRed
	StartColor     = ColorGreen
	TimestampColor = ColorGreen
	NumberColor    = ColorBlue
)

var terminalEscapeSequences = regexp.MustCompile(`(?i)([\x9b\x1b]\[)[0-?]*[ -\/]*[@-~]`)

func Color(colorCode string, str string) string {
	lines := strings.Split(str, "\n")
	for i, line := range lines {
		lines[i] = fmt.Sprintf("\x1b[1m\x1b[%sm%s\x1b[39m\x1b[22m", colorCode, line)
	}
	return strings.Join(lines, "\n")
}

func StripEscapeSequences(str string) string {
	return terminalEscapeSequences.ReplaceAllString(str, "")
}

func Colorize(text string) string {
	// Replicates colorize in typescript: splits text by escape sequences, colorizes plain text fragments, and joins them back.
	matches := terminalEscapeSequences.FindAllStringIndex(text, -1)
	if len(matches) == 0 {
		return colorizePlainText(text)
	}

	var builder strings.Builder
	lastIdx := 0
	for _, match := range matches {
		builder.WriteString(colorizePlainText(text[lastIdx:match[0]]))
		builder.WriteString(text[match[0]:match[1]])
		lastIdx = match[1]
	}
	builder.WriteString(colorizePlainText(text[lastIdx:]))
	return builder.String()
}

func colorizePlainText(text string) string {
	// Colorize numbers inside plain text
	// regex matches digits (optionally with dot-separated sub-parts) not surrounded by alphanumeric characters or dashes/underscores
	numRegex := regexp.MustCompile(`(?i)(?:^|[^A-Za-z0-9_\-\.])[0-9]+(?:\.[0-9]+)*(?:$|[^A-Za-z0-9_\-\.])`)
	
	return numRegex.ReplaceAllStringFunc(text, func(m string) string {
		// Find start and end of actual digits
		start := 0
		for start < len(m) && (m[start] < '0' || m[start] > '9') {
			start++
		}
		end := len(m)
		for end > start && (m[end-1] < '0' || m[end-1] > '9') && m[end-1] != '.' {
			end--
		}
		if start >= end {
			return m
		}
		digits := m[start:end]
		return m[:start] + Color(NumberColor, digits) + m[end:]
	})
}

// Plain Text Logger Implementation
type plainLogger struct {
	writer   io.Writer
	minLevel LogLevel
}

func NewPlainLogger(w io.Writer, minLevel LogLevel) Log {
	return &plainLogger{writer: w, minLevel: minLevel}
}

func (p *plainLogger) Write(text string, level LogLevel) {
	if level < p.minLevel {
		return
	}
	formattedText := fmt.Sprintf("[%s] %s\n", time.Now().Format(time.RFC3339), StripEscapeSequences(text))
	p.writer.Write([]byte(formattedText))
}

func (p *plainLogger) Raw(text string, level LogLevel) {
	if level < p.minLevel {
		return
	}
	p.writer.Write([]byte(text))
}

func (p *plainLogger) Start(text string, level LogLevel) int64 {
	if level >= p.minLevel {
		formattedText := fmt.Sprintf("[%s] Start: %s\n", time.Now().Format(time.RFC3339), StripEscapeSequences(text))
		p.writer.Write([]byte(formattedText))
	}
	return time.Now().UnixNano() / int64(time.Millisecond)
}

func (p *plainLogger) Stop(text string, start int64, level LogLevel) {
	if level < p.minLevel {
		return
	}
	duration := (time.Now().UnixNano() / int64(time.Millisecond)) - start
	formattedText := fmt.Sprintf("[%s] Stop (%d ms): %s\n", time.Now().Format(time.RFC3339), duration, StripEscapeSequences(text))
	p.writer.Write([]byte(formattedText))
}

// JSON Logger Implementation
type jsonLogger struct {
	writer   io.Writer
	minLevel LogLevel
}

func NewJSONLogger(w io.Writer, minLevel LogLevel) Log {
	return &jsonLogger{writer: w, minLevel: minLevel}
}

func (j *jsonLogger) Write(text string, level LogLevel) {
	if level < j.minLevel {
		return
	}
	j.writeEvent(LogEvent{
		Type:      "text",
		Level:     level,
		Timestamp: time.Now().UnixNano() / int64(time.Millisecond),
		Text:      text + "\n",
	})
}

func (j *jsonLogger) Raw(text string, level LogLevel) {
	if level < j.minLevel {
		return
	}
	j.writeEvent(LogEvent{
		Type:      "raw",
		Level:     level,
		Timestamp: time.Now().UnixNano() / int64(time.Millisecond),
		Text:      text,
	})
}

func (j *jsonLogger) Start(text string, level LogLevel) int64 {
	now := time.Now().UnixNano() / int64(time.Millisecond)
	if level >= j.minLevel {
		j.writeEvent(LogEvent{
			Type:      "start",
			Level:     level,
			Timestamp: now,
			Text:      text,
		})
	}
	return now
}

func (j *jsonLogger) Stop(text string, start int64, level LogLevel) {
	if level < j.minLevel {
		return
	}
	j.writeEvent(LogEvent{
		Type:           "stop",
		Level:          level,
		Timestamp:      time.Now().UnixNano() / int64(time.Millisecond),
		Text:           text,
		StartTimestamp: start,
	})
}

func (j *jsonLogger) writeEvent(e LogEvent) {
	data, _ := json.Marshal(e)
	j.writer.Write(append(data, '\n'))
}

// Terminal Logger Implementation (supports colorized interactive logs)
type terminalLogger struct {
	writer       io.Writer
	minLevel     LogLevel
	sessionStart int64
}

func NewTerminalLogger(w io.Writer, minLevel LogLevel) Log {
	return &terminalLogger{
		writer:       w,
		minLevel:     minLevel,
		sessionStart: time.Now().UnixNano() / int64(time.Millisecond),
	}
}

func (t *terminalLogger) Write(text string, level LogLevel) {
	if level < t.minLevel {
		return
	}
	elapsed := (time.Now().UnixNano() / int64(time.Millisecond)) - t.sessionStart
	t.writer.Write([]byte(fmt.Sprintf("[%s] %s\r\n", Color(TimestampColor, strconv.FormatInt(elapsed, 10)+" ms"), Colorize(text))))
}

func (t *terminalLogger) Raw(text string, level LogLevel) {
	if level < t.minLevel {
		return
	}
	t.writer.Write([]byte(text))
}

func (t *terminalLogger) Start(text string, level LogLevel) int64 {
	now := time.Now().UnixNano() / int64(time.Millisecond)
	if level >= t.minLevel {
		elapsed := now - t.sessionStart
		t.writer.Write([]byte(fmt.Sprintf("[%s] Start: %s\r\n", Color(TimestampColor, strconv.FormatInt(elapsed, 10)+" ms"), Colorize(text))))
	}
	return now
}

func (t *terminalLogger) Stop(text string, start int64, level LogLevel) {
	if level < t.minLevel {
		return
	}
	now := time.Now().UnixNano() / int64(time.Millisecond)
	elapsed := now - t.sessionStart
	duration := now - start
	t.writer.Write([]byte(fmt.Sprintf("[%s] Stop (%d ms): %s\r\n", Color(TimestampColor, strconv.FormatInt(elapsed, 10)+" ms"), duration, Colorize(text))))
}
