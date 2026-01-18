package config

import "log/slog"

const (
	LoggerCtxKey = "logger"
)

var (
	Debug bool
)

func GetLogLevel() slog.Level {
	if Debug {
		return slog.LevelDebug
	}
	return slog.LevelInfo
}
