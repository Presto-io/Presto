package main

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

func TestInitLoggerWithLogFileDoesNotPanic(t *testing.T) {
	originalLogger := logger
	originalLogFile := loggerLogFile
	originalLogFilePath := logFilePath
	originalVerbose := verbose
	defer func() {
		if loggerLogFile != nil && loggerLogFile != originalLogFile {
			_ = loggerLogFile.Close()
		}
		logger = originalLogger
		loggerLogFile = originalLogFile
		logFilePath = originalLogFilePath
		verbose = originalVerbose
		if logger != nil {
			slog.SetDefault(logger)
		}
	}()

	logFilePath = filepath.Join(t.TempDir(), "presto.log")
	verbose = true
	logger = nil
	loggerLogFile = nil

	initLogger()

	if logger == nil {
		t.Fatal("logger was not initialized")
	}
	if loggerLogFile == nil {
		t.Fatal("loggerLogFile was not initialized")
	}
	logger.Info("[test] log write")
	_ = loggerLogFile.Close()

	data, err := os.ReadFile(logFilePath)
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected log file to contain logger output")
	}
}
