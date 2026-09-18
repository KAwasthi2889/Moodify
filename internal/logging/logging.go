package logging

import (
	"log/slog"
	"os"
)

// Setup initializes the global slog logger with JSON output to stdout.
// This produces structured logs suitable for production consumption
// and future OpenTelemetry integration.
func Setup() {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	slog.SetDefault(slog.New(handler))
}
