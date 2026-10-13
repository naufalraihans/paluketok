package logger

import (
	"log/slog"
	"os"
	"strings"
)

// Setup memasang logger JSON sebagai slog default. Level dari LOG_LEVEL.
func Setup(env string) *slog.Logger {
	lvl := slog.LevelInfo
	switch strings.ToLower(os.Getenv("LOG_LEVEL")) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	}

	var h slog.Handler
	if env == "development" {
		h = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: lvl})
	} else {
		h = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl})
	}
	l := slog.New(h)
	slog.SetDefault(l)
	return l
}
