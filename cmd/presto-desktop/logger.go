package main

import (
	"io"
	"log/slog"
	"os"
	"strings"

	"gopkg.in/natefinch/lumberjack.v2"
)

func initLogger() {
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}

	var writer io.Writer = os.Stderr

	if logFilePath != "" {
		loggerLogFile = &lumberjack.Logger{
			Filename:   logFilePath,
			MaxSize:    10,
			MaxBackups: 5,
			MaxAge:     0,
			Compress:   false,
			LocalTime:  true,
		}

		writer = io.MultiWriter(os.Stderr, loggerLogFile)

		logger.Info("[logger] log rotation enabled",
			"max_size_mb", 10,
			"max_backups", 5,
			"log_file", logFilePath)
	}

	opts := &slog.HandlerOptions{
		Level:       level,
		ReplaceAttr: sanitizeLogAttributes,
	}
	handler := slog.NewTextHandler(writer, opts)

	logger = slog.New(handler)
	slog.SetDefault(logger)

	logger.Info("[presto] logger initialized",
		"verbose", verbose,
		"log_file", logFilePath,
		"level", level.String())
}

func closeLogger() {
	if loggerLogFile != nil {
		logger.Info("[presto] shutting down logger")
		loggerLogFile.Close()
	}
}

func sanitizeLogAttributes(groups []string, a slog.Attr) slog.Attr {
	if a.Value.Kind() == slog.KindString {
		value := a.Value.String()
		home := homeDir()
		if home != "" && strings.Contains(value, home) {
			value = strings.ReplaceAll(value, home, "~")
			a.Value = slog.StringValue(value)
		}
	}
	return a
}

func homeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home
}
