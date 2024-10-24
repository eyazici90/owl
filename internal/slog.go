package internal

import (
	"io"
	"log/slog"
	"strings"
)

func SetUpSlog(wr io.Writer, level slog.Level) {
	opts := &slog.HandlerOptions{
		Level: level,
	}
	h := slog.NewTextHandler(wr, opts)
	sl := slog.New(h)
	slog.SetDefault(sl)
}

func ParseLevel(s string) slog.Level {
	switch strings.ToLower(s) {
	case "error":
		return slog.LevelError
	case "warn":
		return slog.LevelWarn
	case "info":
		return slog.LevelInfo
	case "debug":
		return slog.LevelDebug
	default:
		return slog.LevelInfo
	}
}
