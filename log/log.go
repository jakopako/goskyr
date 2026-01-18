package log

import (
	"context"
	"log/slog"
	"os"

	"github.com/jakopako/goskyr/config"
)

func InitializeDefaultLogger() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: config.GetLogLevel()}))
	slog.SetDefault(logger)
}

func ContextWithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, config.LoggerCtxKey, logger)
}

func LoggerFromContext(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(config.LoggerCtxKey).(*slog.Logger); ok {
		return logger
	}
	return slog.Default()
}
