package logger

import (
	"log/slog"
	"os"
	"strings"
)

func New(level string) *slog.Logger {
	var parsedLevel slog.Level
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		parsedLevel = slog.LevelDebug
	case "warn":
		parsedLevel = slog.LevelWarn
	case "error":
		parsedLevel = slog.LevelError
	default:
		parsedLevel = slog.LevelInfo
	}

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: parsedLevel})
	return slog.New(handler)
}
