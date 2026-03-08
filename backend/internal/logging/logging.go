package logging

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

func NewLogger(appEnv string, writer io.Writer) *slog.Logger {
	if writer == nil {
		writer = os.Stdout
	}

	options := &slog.HandlerOptions{
		Level: levelFromAppEnv(appEnv),
	}

	if isLocalLikeEnv(appEnv) {
		return slog.New(slog.NewTextHandler(writer, options))
	}
	return slog.New(slog.NewJSONHandler(writer, options))
}

func Init(appEnv string) {
	slog.SetDefault(NewLogger(appEnv, os.Stdout))
}

func Fatal(message string, args ...any) {
	slog.Error(message, args...)
	os.Exit(1)
}

func levelFromAppEnv(appEnv string) slog.Leveler {
	if isLocalLikeEnv(appEnv) {
		return slog.LevelDebug
	}
	return slog.LevelInfo
}

func isLocalLikeEnv(appEnv string) bool {
	normalized := strings.TrimSpace(strings.ToLower(appEnv))
	return normalized == "" || normalized == "local" || normalized == "test"
}
