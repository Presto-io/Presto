package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	macOptions "github.com/wailsapp/wails/v2/pkg/options/mac"
	windowsOptions "github.com/wailsapp/wails/v2/pkg/options/windows"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/mrered/presto/internal/api"
	"github.com/mrered/presto/internal/appdata"
	"github.com/mrered/presto/internal/preview"
	"github.com/mrered/presto/internal/template"
	"github.com/mrered/presto/internal/typst"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	version           string
	startupURL        string
	logger            *slog.Logger
	verbose           bool
	logFilePath       string
	migrateLegacy     bool
	downloadTemplates bool
	loggerLogFile     *lumberjack.Logger
)

func init() {
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose (debug) logging")
	flag.BoolVar(&verbose, "v", false, "Enable verbose (debug) logging (shorthand)")
	flag.StringVar(&logFilePath, "log-file", "", "Write logs to file path")
	flag.BoolVar(&migrateLegacy, "migrate-legacy-data", false, "Migrate legacy app data and exit")
	flag.BoolVar(&downloadTemplates, "download-templates", false, "Download official templates and exit")
}

//go:embed all:build
var assets embed.FS

type App struct {
	ctx             context.Context
	manager         *template.Manager
	compiler        *typst.Compiler
	registry        *template.RegistryCache
	capabilities    ReleaseCapabilities
	previewService  *preview.Service
	previewRunner   *previewRunner
	saveMenuItem    *menu.MenuItem
	exportMenuItem  *menu.MenuItem
	hasDirtyContent bool
	currentFilename string
	externalFilesMu sync.Mutex
	startupFiles    []OpenFilesItem
	frontendReady   bool
	fileOpenReady   bool
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

func NewApp(manager *template.Manager, compiler *typst.Compiler, registry *template.RegistryCache, capabilities ReleaseCapabilities, previewService *preview.Service, previewRunner *previewRunner) *App {
	return &App{
		manager:        manager,
		compiler:       compiler,
		registry:       registry,
		capabilities:   normalizeReleaseCapabilities(capabilities),
		previewService: previewService,
		previewRunner:  previewRunner,
	}
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
		a.markFrontendReady()
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
		capabilities := a.releaseCapabilities()
		if capabilities.FirstLaunchBootstrap || capabilities.TemplateAutoUpdate {
			a.checkFirstLaunch()
		} else {
			logger.Info("[startup] online template bootstrap disabled by release channel", "channel", capabilities.ReleaseChannel)
		}
		if capabilities.AppUpdateCheck {
			go a.CheckStartupUpdate()
		}
	}()
}

func main() {
	flag.Parse()
	initLogger()
	dirs, err := appdata.ResolveDirs()
	if err != nil {
		log.Fatal("failed to resolve app data directories: ", err)
	}
	if migrateLegacy {
		migrateLegacyDataAndExit(dirs)
		return
	}
	capabilities := currentReleaseCapabilities()
	if downloadTemplates {
		if !capabilities.OnlineRegistry || !capabilities.OnlineTemplateStore {
			log.Fatal("download-templates is disabled in this release channel")
		}
		downloadTemplatesAndExit(dirs)
		return
	}
	if result, err := appdata.MigrateLegacyOnce(dirs); err != nil {
		logger.Warn("[presto] failed to migrate legacy app data", "error", err)
	} else if result.Attempted && len(result.Migrated) > 0 {
		logger.Info("[presto] migrated legacy app data", "items", strings.Join(result.Migrated, ","), "conflicts", strings.Join(result.Conflicts, ","))
	}
	if err := dirs.Ensure(); err != nil {
		log.Fatal("failed to create app data directories: ", err)
	}
	prestoDir := dirs.DataDir
	templatesDir := dirs.TemplatesDir()
	if err := os.MkdirAll(templatesDir, 0755); err != nil {
		log.Fatal("failed to create templates directory: ", err)
	}
	fontsDir := dirs.FontsDir()
	if err := os.MkdirAll(fontsDir, 0755); err != nil {
		log.Fatal("failed to create fonts directory: ", err)
	}
	fontPaths := appdata.ResolveFontPaths(fontsDir)
	if err := appdata.MarkGenerated(prestoDir); err != nil {
		logger.Warn("[presto] failed to mark generated app data", "error", err)
	}
	exeDir := ""
	if exePath, err := os.Executable(); err == nil {
		if resolved, err := filepath.EvalSymlinks(exePath); err == nil {
			exePath = resolved
		}
		exeDir = filepath.Dir(exePath)
	}
	if err := validatePortablePackagedRuntimes(capabilities, exeDir, runtime.GOOS, runtime.GOARCH); err != nil {
		log.Fatal(err)
	}
	typstBin := findTypstBinaryFrom(exeDir, dirs.DataDir, runtime.GOOS, exec.LookPath)
	logger.Info("[presto] using typst", "path", typstBin)
	tinymistBin := findTinymistBinaryFrom(exeDir, dirs.DataDir, runtime.GOOS, runtime.GOARCH, exec.LookPath)
	logger.Info("[presto] using tinymist", "path", tinymistBin)
	builtinTemplatesDir := ""
	if capabilities.PackagedRuntimes || capabilities.ReleaseChannel == "portable" {
		builtinTemplatesDir = template.ResolveBuiltinTemplatesDir(exeDir, runtime.GOOS)
	}
	manager := template.NewManagerWithBuiltin(templatesDir, builtinTemplatesDir)
	var registry *template.RegistryCache
	if capabilities.OnlineRegistry {
		registry = template.NewRegistryCache(dirs.CacheDir)
		registry.RefreshAsync()
	} else {
		logger.Info("[presto] online registry disabled by release channel", "channel", capabilities.ReleaseChannel)
	}
	// SEC-40: Use os temp dir instead of $HOME to restrict file access
	compiler := typst.NewCompilerWithRoot(os.TempDir())
	compiler.BinPath = typstBin
	compiler.FontPaths = fontPaths
	compiler.AvailableFonts = compiler.ListFonts()
	apiHandler := api.NewServer(api.ServerOptions{
		TemplatesDir:        templatesDir,
		BuiltinTemplatesDir: builtinTemplatesDir,
		TypstBin:            typstBin,
		FontPaths:           fontPaths,
		Registry:            registry,
		Capabilities:        toAPIReleaseCapabilities(capabilities),
	})
	frontendFS, _ := fs.Sub(assets, "build")
	handler := &spaFallbackHandler{api: apiHandler, assets: frontendFS}
	previewService := preview.NewService()
	previewRunner := newPreviewRunner(previewService, tinymistBin)
	app := NewApp(manager, compiler, registry, capabilities, previewService, previewRunner)
	var appMenu *menu.Menu
	if runtime.GOOS != "windows" {
		appMenu = buildMenu(app)
	}
	logger.Debug("[url-scheme] os.Args", "args", os.Args)
	for _, arg := range os.Args[1:] {
		if strings.HasPrefix(arg, "presto://") {
			startupURL = arg
			logger.Debug("[url-scheme] captured startup URL", "url", arg)
			break
		}
	}
	app.dispatchOrQueueExternalFiles(os.Args[1:])
	err = wails.Run(&options.App{
		Title:            "Presto",
		Width:            1280,
		Height:           800,
		MinWidth:         800,
		MinHeight:        600,
		Frameless:        runtime.GOOS == "windows",
		BackgroundColour: options.NewRGB(26, 27, 38),
		AssetServer: &assetserver.Options{
			Assets:  frontendFS,
			Handler: handler,
		},
		DragAndDrop: &options.DragAndDrop{
			EnableFileDrop: true,
		},
		Menu:      appMenu,
		OnStartup: app.startup,
		OnBeforeClose: func(ctx context.Context) (prevent bool) {
			if !app.hasDirtyContent {
				if app.previewRunner != nil {
					_ = app.previewRunner.stop()
				}
				return false
			}
			wailsRuntime.EventsEmit(ctx, "app:request-close")
			return true
		},
		Bind: []interface{}{app},
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId: "com.mrered.presto",
			OnSecondInstanceLaunch: func(data options.SecondInstanceData) {
				logger.Debug("[single-instance] second instance args", "args", data.Args)
				for _, arg := range data.Args {
					if strings.HasPrefix(arg, "presto://") {
						go app.handlePrestoURL(arg)
						return
					}
				}
				app.dispatchOrQueueExternalFiles(data.Args)
			},
		},
		Windows: &windowsOptions.Options{
			DisableWindowIcon:   false,
			Theme:               windowsOptions.Dark,
			BackdropType:        windowsOptions.None,
			WebviewUserDataPath: dirs.WebViewDataDir(),
			CustomTheme: &windowsOptions.ThemeSettings{
				DarkModeTitleBar:          windowsOptions.RGB(26, 27, 38),
				DarkModeTitleBarInactive:  windowsOptions.RGB(31, 33, 51),
				DarkModeTitleText:         windowsOptions.RGB(224, 228, 247),
				DarkModeTitleTextInactive: windowsOptions.RGB(86, 95, 137),
				DarkModeBorder:            windowsOptions.RGB(55, 59, 86),
				DarkModeBorderInactive:    windowsOptions.RGB(42, 45, 68),
			},
		},
		Mac: &macOptions.Options{
			TitleBar: macOptions.TitleBarHiddenInset(),
			About: &macOptions.AboutInfo{
				Title:   "Presto",
				Message: "Markdown → Typst → PDF",
			},
			OnFileOpen: func(filePath string) {
				logger.Debug("[file-open] macOS open file", "path", filePath)
				app.dispatchOrQueueExternalFiles([]string{filePath})
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

func validatePortablePackagedRuntimes(capabilities ReleaseCapabilities, exeDir string, goos string, goarch string) error {
	if !capabilities.PackagedRuntimes {
		return nil
	}
	if findPackagedTypstBinary(exeDir, goos) == "" {
		return fmt.Errorf("portable packaged runtime missing: typst")
	}
	if findPackagedTinymistBinary(exeDir, goos, goarch) == "" {
		return fmt.Errorf("portable packaged runtime missing: tinymist")
	}
	return nil
}
