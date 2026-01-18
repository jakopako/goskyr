package config

import "log/slog"

var (
	Debug bool
)

func GetLogLevel() slog.Level {
	if Debug {
		return slog.LevelDebug
	}
	return slog.LevelInfo
}
