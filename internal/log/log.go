package log

import (
	"context"
	"log/slog"
	"os"
)

var (
	Debug bool
)

const (
	LoggerCtxKey = "logger"
)

func GetLogLevel() slog.Level {
	if Debug {
		return slog.LevelDebug
	}
	return slog.LevelInfo
}

func InitializeDefaultLogger() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: GetLogLevel()}))
	slog.SetDefault(logger)
}

func ContextWithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, LoggerCtxKey, logger)
}

func LoggerFromContext(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(LoggerCtxKey).(*slog.Logger); ok {
		return logger
	}
	return slog.Default()
}
