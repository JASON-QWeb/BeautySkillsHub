package logging

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestNewLogger_UsesTextHandlerForLocal(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := NewLogger("local", &buf)
	logger.Info("local message", "component", "api")

	output := buf.String()
	if strings.Contains(output, "{\"time\"") {
		t.Fatalf("expected text output for local env, got %q", output)
	}
	if !strings.Contains(output, "level=INFO") {
		t.Fatalf("expected text handler fields, got %q", output)
	}
	if !strings.Contains(output, "component=api") {
		t.Fatalf("expected structured text attribute, got %q", output)
	}
}

func TestNewLogger_UsesJSONHandlerOutsideLocal(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := NewLogger("production", &buf)
	logger.Info("prod message", "component", "api")

	output := buf.String()
	if !strings.Contains(output, "\"level\":\"INFO\"") {
		t.Fatalf("expected json level field, got %q", output)
	}
	if !strings.Contains(output, "\"component\":\"api\"") {
		t.Fatalf("expected json attribute field, got %q", output)
	}
}

func TestLevelFromAppEnv_UsesDebugForLocal(t *testing.T) {
	t.Parallel()

	if levelFromAppEnv("local") != slog.LevelDebug {
		t.Fatalf("expected local env to use debug level")
	}
	if levelFromAppEnv("production") != slog.LevelInfo {
		t.Fatalf("expected production env to use info level")
	}
}
