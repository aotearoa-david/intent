package logging

import (
	"log/slog"
	"os"
)

// NewLogger configures a JSON structured logger for the service.
func NewLogger() *slog.Logger {
	level := slog.LevelInfo
	if env := os.Getenv("LOG_LEVEL"); env != "" {
		var parsed slog.Level
		if err := parsed.UnmarshalText([]byte(env)); err == nil {
			level = parsed
		}
	}

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	return slog.New(handler)
}
