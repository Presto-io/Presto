package main

import (
	"context"
	"embed"
	"flag"
	"io/fs"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	macOptions "github.com/wailsapp/wails/v2/pkg/options/mac"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/mrered/presto/internal/api"
	"github.com/mrered/presto/internal/template"
	"github.com/mrered/presto/internal/typst"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	version    string
	startupURL string
	logger        *slog.Logger
	verbose       bool
	logFilePath   string
	loggerLogFile *lumberjack.Logger
)

func init() {
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose (debug) logging")
	flag.BoolVar(&verbose, "v", false, "Enable verbose (debug) logging (shorthand)")
	flag.StringVar(&logFilePath, "log-file", "", "Write logs to file path")
}

//go:embed all:build
var assets embed.FS

type App struct {
	ctx            context.Context
	manager        *template.Manager
	compiler       *typst.Compiler
	registry       *template.RegistryCache
	saveMenuItem   *menu.MenuItem
	exportMenuItem *menu.MenuItem
}

type spaFallbackHandler struct {
	api    http.Handler
	assets fs.FS
}

func (h *spaFallbackHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/mock/") {
		h.api.ServeHTTP(w, r)
		return
	}
	cleanPath := strings.TrimPrefix(r.URL.Path, "/")
	if cleanPath != "" {
		if data, err := fs.ReadFile(h.assets, cleanPath+".html"); err == nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write(data)
			return
		}
	}
	if data, err := fs.ReadFile(h.assets, "index.html"); err == nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(data)
		return
	}
	http.NotFound(w, r)
}

func NewApp(manager *template.Manager, compiler *typst.Compiler, registry *template.RegistryCache) *App {
	return &App{manager: manager, compiler: compiler, registry: registry}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	template.CleanupTmpDownloadFiles()
	wailsRuntime.OnFileDrop(ctx, func(x, y int, paths []string) {
		items := a.readFilePaths(paths)
		if len(items) > 0 {
			wailsRuntime.EventsEmit(ctx, "native-file-drop", items)
		}
	})
	frontendReady := make(chan struct{}, 1)
	wailsRuntime.EventsOnce(ctx, "frontend:ready", func(optionalData ...interface{}) {
		select {
		case frontendReady <- struct{}{}:
		default:
		}
	})
	go func() {
		select {
		case <-frontendReady:
			logger.Info("[startup] frontend ready, checking first launch")
		case <-time.After(5 * time.Second):
			logger.Warn("[startup] frontend ready timeout, proceeding anyway")
		}
		a.checkFirstLaunch()
	}()
}

func main() {
	flag.Parse()
	initLogger()
	if len(os.Args) > 1 && os.Args[1] == "--download-templates" {
		downloadTemplatesAndExit()
		return
	}
	// SEC-44: Check os.UserHomeDir error
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal("failed to get home directory: ", err)
	}
	prestoDir := filepath.Join(home, ".presto")
	templatesDir := filepath.Join(prestoDir, "templates")
	if err := os.MkdirAll(templatesDir, 0755); err != nil {
		log.Fatal("failed to create templates directory: ", err)
	}
	manager := template.NewManager(templatesDir)
	typstBin := findTypstBinary()
	logger.Info("[presto] using typst", "path", typstBin)
	registry := template.NewRegistryCache(prestoDir)
	registry.RefreshAsync()
	// SEC-40: Use os temp dir instead of $HOME to restrict file access
	compiler := typst.NewCompilerWithRoot(os.TempDir())
	compiler.BinPath = typstBin
	apiHandler := api.NewServer(api.ServerOptions{
		TemplatesDir: templatesDir,
		TypstBin:     typstBin,
		Registry:     registry,
	})
	frontendFS, _ := fs.Sub(assets, "build")
	handler := &spaFallbackHandler{api: apiHandler, assets: frontendFS}
	app := NewApp(manager, compiler, registry)
	appMenu := buildMenu(app)
	logger.Debug("[url-scheme] os.Args", "args", os.Args)
	for _, arg := range os.Args[1:] {
		if strings.HasPrefix(arg, "presto://") {
			startupURL = arg
			logger.Debug("[url-scheme] captured startup URL", "url", arg)
			break
		}
	}
	err = wails.Run(&options.App{
		Title:     "Presto",
		Width:     1280,
		Height:    800,
		MinWidth:  800,
		MinHeight: 600,
		AssetServer: &assetserver.Options{
			Assets:  frontendFS,
			Handler: handler,
		},
		DragAndDrop: &options.DragAndDrop{
			EnableFileDrop: true,
		},
		Menu:      appMenu,
		OnStartup: app.startup,
		Bind:      []interface{}{app},
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId: "com.mrered.presto",
			OnSecondInstanceLaunch: func(data options.SecondInstanceData) {
				logger.Debug("[url-scheme] second instance args", "args", data.Args)
			},
		},
		Mac: &macOptions.Options{
			TitleBar: macOptions.TitleBarHiddenInset(),
			About: &macOptions.AboutInfo{
				Title:   "Presto",
				Message: "Markdown → Typst → PDF",
			},
			OnUrlOpen: func(url string) {
				logger.Debug("[url-scheme] OnUrlOpen", "url", url)
				if app.ctx != nil {
					go app.handlePrestoURL(url)
				} else {
					startupURL = url
				}
			},
		},
	})
	if err != nil {
		println("Error:", err.Error())
	}
}
