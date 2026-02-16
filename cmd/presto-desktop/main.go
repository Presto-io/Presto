package main

import (
	"context"
	"embed"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	"github.com/mrered/presto/internal/api"
)

//go:embed all:build
var assets embed.FS

// App struct for Wails bindings (future Path B)
type App struct {
	ctx context.Context
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func main() {
	home, _ := os.UserHomeDir()
	templatesDir := filepath.Join(home, ".presto", "templates")
	os.MkdirAll(templatesDir, 0755)

	// Reuse existing API server as HTTP handler for /api/* routes
	apiHandler := api.NewServer(templatesDir, "")

	// Strip "build" prefix from embedded FS so files are at root
	frontendFS, _ := fs.Sub(assets, "build")

	app := NewApp()

	err := wails.Run(&options.App{
		Title:     "Presto",
		Width:     1280,
		Height:    800,
		MinWidth:  800,
		MinHeight: 600,
		AssetServer: &assetserver.Options{
			Assets:  frontendFS,
			Handler: apiHandler,
		},
		OnStartup: app.startup,
		Bind: []interface{}{
			app,
		},
	})
	if err != nil {
		println("Error:", err.Error())
	}
}
