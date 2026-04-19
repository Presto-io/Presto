package main

import (
	"fmt"
	"os"
	"runtime"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/mrered/presto/internal/api"
)

func (a *App) GetVersion() string {
	if version == "" {
		return "dev"
	}
	return version
}

func (a *App) GetPlatform() string {
	return runtime.GOOS
}

func (a *App) SetWindowTitle(title string) {
	if runtime.GOOS == "windows" {
		wailsRuntime.WindowSetTitle(a.ctx, title)
	}
}

func (a *App) QuitApp() {
	wailsRuntime.Quit(a.ctx)
}

func (a *App) ImportBatchZip(filePath string) (*api.BatchImportResult, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read ZIP failed: %w", err)
	}
	return api.ProcessBatchZip(data, a.manager, a.registry)
}
