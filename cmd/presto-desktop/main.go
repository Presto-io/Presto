package main

import (
	"context"
	"embed"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	macOptions "github.com/wailsapp/wails/v2/pkg/options/mac"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/mrered/presto/internal/api"
	"github.com/mrered/presto/internal/template"
	"github.com/mrered/presto/internal/typst"
	"gopkg.in/natefinch/lumberjack.v2"
)

// version is set at build time via -ldflags "-X main.version=..."
var version string

// startupURL holds a presto:// URL passed via os.Args on cold start.
var startupURL string

// Logging configuration
var (
	logger        *slog.Logger
	verbose       bool
	logFilePath   string
	loggerLogFile *lumberjack.Logger // for cleanup on shutdown
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

// spaFallbackHandler wraps the API handler to support prerendered SvelteKit routes.
// When Wails can't find a static asset, it calls this handler. We try:
// 1. path + ".html" (prerendered routes like /showcase/editor → showcase/editor.html)
// 2. Forward /api/* and /mock/* to the API handler
// 3. Serve index.html as SPA fallback
type spaFallbackHandler struct {
	api    http.Handler
	assets fs.FS
}

func (h *spaFallbackHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// API and mock routes → forward directly
	if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/mock/") {
		h.api.ServeHTTP(w, r)
		return
	}

	// Try .html suffix for prerendered routes
	cleanPath := strings.TrimPrefix(r.URL.Path, "/")
	if cleanPath != "" {
		htmlPath := cleanPath + ".html"
		if data, err := fs.ReadFile(h.assets, htmlPath); err == nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write(data)
			return
		}
	}

	// SPA fallback → index.html
	if data, err := fs.ReadFile(h.assets, "index.html"); err == nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(data)
		return
	}

	http.NotFound(w, r)
}

type OpenFileResult struct {
	Content string `json:"content"`
	Dir     string `json:"dir"`
}

// OpenFilesItem represents a single file selected via the multi-file dialog.
type OpenFilesItem struct {
	Name    string `json:"name"`
	Content string `json:"content"` // text for md/txt; empty for zip
	Dir     string `json:"dir"`
	IsZip   bool   `json:"isZip"`
	Path    string `json:"path,omitempty"` // absolute path (zip only, for Wails binding)
}

func NewApp(manager *template.Manager, compiler *typst.Compiler, registry *template.RegistryCache) *App {
	return &App{manager: manager, compiler: compiler, registry: registry}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Clean up temporary download files from previous session
	template.CleanupTmpDownloadFiles()

	// Native file drop: receive absolute paths and forward to frontend
	wailsRuntime.OnFileDrop(ctx, func(x, y int, paths []string) {
		items := a.readFilePaths(paths)
		if len(items) > 0 {
			wailsRuntime.EventsEmit(ctx, "native-file-drop", items)
		}
	})

	// Check if this is first launch (no templates installed)
	go a.checkFirstLaunch()
}

// checkFirstLaunch detects if this is the first launch (no templates installed)
// and triggers automatic download of official templates.
func (a *App) checkFirstLaunch() {
	logger.Debug("[first-launch] starting check", "registry_available", a.registry.Load() != nil)

	templates, err := a.manager.List()
	if err != nil {
		logger.Error("[first-launch] failed to list templates", "error", err)
		return
	}

	// First launch if no templates installed
	if len(templates) == 0 {
		logger.Info("[first-launch] first launch detected, starting default template download")
		a.downloadDefaultTemplates()
		return
	}

	// Templates already installed - check for updates
	logger.Info("[first-launch] templates already installed, checking for updates", "count", len(templates))
	go a.checkTemplateUpdates(templates)
}

// downloadDefaultTemplates downloads all official templates concurrently.
// Emits events for frontend to track progress.
func (a *App) downloadDefaultTemplates() {
	reg := a.registry.Load()
	if reg == nil {
		logger.Warn("[first-launch] registry not available, skipping default download")
		a.emitFirstLaunchError("无法获取模板列表")
		return
	}

	// Filter official templates
	var officialTemplates []string
	for _, entry := range reg.Templates {
		if entry.Trust == "official" {
			officialTemplates = append(officialTemplates, entry.Name)
		}
	}

	if len(officialTemplates) == 0 {
		logger.Warn("[first-launch] no official templates found")
		return
	}

	logger.Info("[first-launch] downloading official templates", "count", len(officialTemplates))

	// Emit start event with template list
	a.emitFirstLaunchStart(officialTemplates)

	// Download concurrently using a semaphore to limit parallelism
	var wg sync.WaitGroup
	sem := make(chan struct{}, 3) // Max 3 concurrent downloads
	var successCount int
	var failureCount int
	var mu sync.Mutex

	for _, name := range officialTemplates {
		wg.Add(1)
		go func(templateName string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			logger.Debug("[first-launch] downloading template", "name", templateName)
			err := a.InstallTemplate(templateName)

			mu.Lock()
			if err != nil {
				failureCount++
				logger.Error("[first-launch] failed to download template", "name", templateName, "error", err)
				a.emitFirstLaunchProgress(templateName, "error", err.Error())
			} else {
				successCount++
				logger.Info("[first-launch] successfully downloaded template", "name", templateName)
				a.emitFirstLaunchProgress(templateName, "success", "")
			}
			mu.Unlock()
		}(name)
	}

	wg.Wait()

	// Emit completion event
	a.emitFirstLaunchComplete(successCount, failureCount)
	logger.Info("[first-launch] download complete", "success", successCount, "failed", failureCount)
}

// emitFirstLaunchStart emits the first-launch:start event to the frontend.
func (a *App) emitFirstLaunchStart(templateNames []string) {
	wailsRuntime.EventsEmit(a.ctx, "first-launch:start", map[string]interface{}{
		"total":     len(templateNames),
		"templates": templateNames,
	})
}

// emitFirstLaunchProgress emits the first-launch:progress event for individual template downloads.
func (a *App) emitFirstLaunchProgress(name string, status string, errorMsg string) {
	wailsRuntime.EventsEmit(a.ctx, "first-launch:progress", map[string]interface{}{
		"name":   name,
		"status": status,
		"error":  errorMsg,
	})
}

// emitFirstLaunchComplete emits the first-launch:complete event when all downloads finish.
func (a *App) emitFirstLaunchComplete(success int, failed int) {
	wailsRuntime.EventsEmit(a.ctx, "first-launch:complete", map[string]int{
		"success": success,
		"failed":  failed,
	})
}

// emitFirstLaunchError emits the first-launch:error event when download process fails.
func (a *App) emitFirstLaunchError(message string) {
	wailsRuntime.EventsEmit(a.ctx, "first-launch:error", map[string]string{"message": message})
}

// checkTemplateUpdates checks if installed templates have updates available and installs them silently.
// This is completely silent - no notifications, no badges, just background updates.
// NOTE: Current implementation deletes old version before installing new version.
// If installation fails, the template will be missing and user needs to manually re-install.
// This is acceptable for v1 - future versions can implement atomic swap.
func (a *App) checkTemplateUpdates(installed []template.InstalledTemplate) {
	logger.Debug("[template-update] starting silent update check", "count", len(installed))

	reg := a.registry.Load()
	if reg == nil {
		logger.Warn("[template-update] registry not available, skipping update check")
		return
	}

	var updatesAvailable []struct {
		name    string
		latest  template.RegistryEntry
		current string
	}

	for _, inst := range installed {
		entry := a.registry.LookupByName(inst.Manifest.Name)
		if entry == nil {
			continue
		}

		// Compare versions
		if entry.Version != inst.Manifest.Version {
			logger.Info("[template-update] update available",
				"name", inst.Manifest.Name,
				"installed", inst.Manifest.Version,
				"latest", entry.Version)
			updatesAvailable = append(updatesAvailable, struct {
				name    string
				latest  template.RegistryEntry
				current string
			}{inst.Manifest.Name, *entry, inst.Manifest.Version})
		}
	}

	if len(updatesAvailable) == 0 {
		logger.Info("[template-update] all templates are up to date")
		return
	}

	logger.Info("[template-update] silently updating templates in background", "count", len(updatesAvailable))

	// Update templates in parallel
	var wg sync.WaitGroup
	sem := make(chan struct{}, 3)
	var successCount, failCount int
	var mu sync.Mutex

	for _, update := range updatesAvailable {
		wg.Add(1)
		go func(name string, entry template.RegistryEntry, oldVersion string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			logger.Info("[template-update] updating template",
				"name", name,
				"old_version", oldVersion,
				"new_version", entry.Version)

			// Delete old version first
			if err := a.manager.Uninstall(name); err != nil {
				logger.Error("[template-update] failed to uninstall old version",
					"name", name,
					"error", err)
				mu.Lock()
				failCount++
				mu.Unlock()
				return
			}

			// Install new version (silent - no events, no notifications)
			if err := a.InstallTemplate(name); err != nil {
				logger.Error("[template-update] failed to update template",
					"name", name,
					"error", err)
				mu.Lock()
				failCount++
				mu.Unlock()
				// NOTE: Old version already deleted, template is now missing
				// User will need to manually re-install (known limitation for v1)
				return
			}

			logger.Info("[template-update] successfully updated template",
				"name", name,
				"version", entry.Version)
			mu.Lock()
			successCount++
			mu.Unlock()
		}(update.name, update.latest, update.current)
	}

	wg.Wait()

	// Log summary (for debugging)
	logger.Info("[template-update] silent update complete", "success", successCount, "failed", failCount)

	// IMPORTANT: No UI notifications, no badges, completely silent
	// Only emit templates-changed event to refresh UI if at least one update succeeded
	if successCount > 0 {
		wailsRuntime.EventsEmit(a.ctx, "templates-changed")
	}
}

// GetStartupURL returns and clears the pending presto:// URL from cold start.
// Called by the frontend on mount to check if the app was launched via URL scheme.
func (a *App) GetStartupURL() string {
	u := startupURL
	startupURL = ""
	logger.Debug("[url-scheme] GetStartupURL called", "url", u)
	return u
}

// handlePrestoURL parses a presto:// URL and routes to the appropriate action.
// Currently supports: presto://install/{template-name}
func (a *App) handlePrestoURL(rawURL string) {
	u, err := url.Parse(rawURL)
	if err != nil {
		log.Printf("[url-scheme] failed to parse URL: %s", rawURL)
		return
	}

	// presto://install/{name} → Host="install", Path="/{name}"
	action := u.Host
	if action != "install" {
		log.Printf("[url-scheme] unsupported action: %s", action)
		return
	}

	templateName := strings.TrimPrefix(u.Path, "/")
	if templateName == "" {
		log.Printf("[url-scheme] missing template name in URL: %s", rawURL)
		return
	}

	log.Printf("[url-scheme] opening template: %s", templateName)

	// Emit event to navigate frontend to the template detail page
	wailsRuntime.EventsEmit(a.ctx, "url-scheme-open-template", templateName)
}

// OpenFile opens a native file dialog and returns the file content and directory.
func (a *App) OpenFile() (*OpenFileResult, error) {
	path, err := wailsRuntime.OpenFileDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "打开 Markdown 文件",
		Filters: []wailsRuntime.FileFilter{
			{DisplayName: "Markdown", Pattern: "*.md;*.markdown;*.txt"},
		},
	})
	if err != nil {
		return nil, err
	}
	if path == "" {
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read failed: %w", err)
	}
	return &OpenFileResult{
		Content: string(data),
		Dir:     filepath.Dir(path),
	}, nil
}

// readFilePaths reads files from absolute paths and returns OpenFilesItem entries.
// Shared by OpenFiles (dialog) and OnFileDrop (native drag-drop).
// ZIP files pass path only (frontend calls ImportBatchZip binding directly).
func (a *App) readFilePaths(paths []string) []OpenFilesItem {
	var items []OpenFilesItem
	for _, p := range paths {
		isZip := strings.HasSuffix(strings.ToLower(p), ".zip")
		item := OpenFilesItem{
			Name:  filepath.Base(p),
			Dir:   filepath.Dir(p),
			IsZip: isZip,
		}
		if isZip {
			// Pass path for Wails binding; no need to base64 encode
			item.Path = p
		} else {
			data, err := os.ReadFile(p)
			if err != nil {
				log.Printf("[desktop] failed to read %s: %v", p, err)
				continue
			}
			item.Content = string(data)
		}
		items = append(items, item)
	}
	return items
}

// OpenFiles opens a native multi-file dialog supporting markdown and ZIP files.
func (a *App) OpenFiles() ([]OpenFilesItem, error) {
	paths, err := wailsRuntime.OpenMultipleFilesDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "打开文件",
		Filters: []wailsRuntime.FileFilter{
			{DisplayName: "支持的文件", Pattern: "*.md;*.markdown;*.txt;*.zip"},
		},
	})
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, nil
	}
	return a.readFilePaths(paths), nil
}

func buildMenu(app *App) *menu.Menu {
	appMenu := menu.NewMenu()
	appMenu.Append(menu.AppMenu())

	// 文件菜单
	fileMenu := appMenu.AddSubmenu("文件")
	fileMenu.AddText("新建", keys.CmdOrCtrl("n"), func(_ *menu.CallbackData) {
		wailsRuntime.EventsEmit(app.ctx, "menu:new")
	})
	fileMenu.AddText("打开文件…", keys.CmdOrCtrl("o"), func(_ *menu.CallbackData) {
		wailsRuntime.EventsEmit(app.ctx, "menu:open")
	})
	app.saveMenuItem = fileMenu.AddText("保存", keys.CmdOrCtrl("s"), func(_ *menu.CallbackData) {
		wailsRuntime.EventsEmit(app.ctx, "menu:save")
	})
	app.saveMenuItem.Disabled = true // MENU-12: disabled when editor is empty
	fileMenu.AddText("另存为…", keys.Combo("s", keys.CmdOrCtrlKey, keys.ShiftKey), func(_ *menu.CallbackData) {
		wailsRuntime.EventsEmit(app.ctx, "menu:saveas")
	})
	app.exportMenuItem = fileMenu.AddText("导出 PDF…", keys.CmdOrCtrl("e"), func(_ *menu.CallbackData) {
		wailsRuntime.EventsEmit(app.ctx, "menu:export")
	})
	app.exportMenuItem.Disabled = true // MENU-12: disabled when editor is empty
	fileMenu.AddText("设置…", keys.CmdOrCtrl(","), func(_ *menu.CallbackData) {
		wailsRuntime.EventsEmit(app.ctx, "menu:settings")
	})
	fileMenu.AddSeparator()
	fileMenu.AddText("最小化", keys.CmdOrCtrl("m"), func(_ *menu.CallbackData) {
		wailsRuntime.WindowMinimise(app.ctx)
	})
	fileMenu.AddText("缩放", nil, func(_ *menu.CallbackData) {
		wailsRuntime.WindowToggleMaximise(app.ctx)
	})
	fileMenu.AddSeparator()
	fileMenu.AddText("退出", keys.CmdOrCtrl("w"), func(_ *menu.CallbackData) {
		wailsRuntime.Quit(app.ctx)
	})

	// 编辑菜单（Wails 内置）
	appMenu.Append(menu.EditMenu())

	// 模板菜单
	templateMenu := appMenu.AddSubmenu("模板")
	templateMenu.AddText("模板商店", nil, func(_ *menu.CallbackData) {
		wailsRuntime.EventsEmit(app.ctx, "menu:store")
	})
	templateMenu.AddText("模板管理…", keys.Combo("t", keys.CmdOrCtrlKey, keys.ShiftKey), func(_ *menu.CallbackData) {
		wailsRuntime.EventsEmit(app.ctx, "menu:templates")
	})

	// 帮助菜单
	helpMenu := appMenu.AddSubmenu("帮助")
	helpMenu.AddText("文档", nil, func(_ *menu.CallbackData) {
		wailsRuntime.BrowserOpenURL(app.ctx, "https://presto.io/docs")
	})
	helpMenu.AddText("关于 Presto", nil, func(_ *menu.CallbackData) {
		app.ShowAboutDialog()
	})
	helpMenu.AddText("检查更新", nil, func(_ *menu.CallbackData) {
		go app.CheckAndNotifyUpdate()
	})

	return appMenu
}

// SaveMarkdown writes markdown content to the given file path.
func (a *App) SaveMarkdown(content string, filePath string) error {
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("save failed: %w", err)
	}
	logger.Info("[desktop] saved markdown", "path", filePath, "bytes", len(content))
 return nil
}

// SaveMarkdownAs opens a native save dialog and writes markdown content to the chosen path.
// Returns the selected path, or ("", nil) if the user cancelled.
func (a *App) SaveMarkdownAs(content string) (string, error) {
	savePath, err := wailsRuntime.SaveFileDialog(a.ctx, wailsRuntime.SaveDialogOptions{
		DefaultFilename: "untitled.md",
		Filters: []wailsRuntime.FileFilter{
			{DisplayName: "Markdown", Pattern: "*.md"},
			{DisplayName: "所有文件", Pattern: "*.*"},
		},
	})
	if err != nil {
		return "", fmt.Errorf("save dialog failed: %w", err)
	}
	if savePath == "" {
		return "", nil // user cancelled
	}
	if err := os.WriteFile(savePath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("write failed: %w", err)
	}
	logger.Info("[desktop] saved markdown as", "path", savePath, "bytes", len(content))
	return savePath, nil
}

// ShowAboutDialog displays a native info dialog with app version and copyright.
func (a *App) ShowAboutDialog() {
	ver := a.GetVersion()
	wailsRuntime.MessageDialog(a.ctx, wailsRuntime.MessageDialogOptions{
		Type:    wailsRuntime.InfoDialog,
		Title:   "关于 Presto",
		Message: fmt.Sprintf("Presto %s\nMarkdown → Typst → PDF\n\n© 2024-2026 Presto", ver),
	})
}

// CheckAndNotifyUpdate checks for updates and notifies the frontend or shows a dialog.
func (a *App) CheckAndNotifyUpdate() {
	info, err := a.CheckForUpdate()
	if err != nil {
		log.Printf("[desktop] update check failed: %v", err)
		return
	}
	if info.HasUpdate {
		wailsRuntime.EventsEmit(a.ctx, "menu:update-available", info)
	} else {
		wailsRuntime.MessageDialog(a.ctx, wailsRuntime.MessageDialogOptions{
			Type:    wailsRuntime.InfoDialog,
			Title:   "检查更新",
			Message: "已是最新版本",
		})
	}
}

// UpdateMenuState enables or disables save/export menu items based on editor content.
// Called by the frontend when editor content changes.
func (a *App) UpdateMenuState(hasContent bool) {
	if a.saveMenuItem != nil {
		a.saveMenuItem.Disabled = !hasContent
	}
	if a.exportMenuItem != nil {
		a.exportMenuItem.Disabled = !hasContent
	}
	wailsRuntime.MenuUpdateApplicationMenu(a.ctx)
}

// CompileSVG compiles typst source to SVG pages via Wails binding,
// bypassing the HTTP layer where Wails WebView strips headers/query params.
func (a *App) CompileSVG(typstSource string, workDir string) ([]string, error) {
	return a.compiler.CompileToSVG(typstSource, workDir)
}

// ImportBatchZip reads a ZIP file from disk and processes it: installs templates
// and extracts markdown files. Bypasses the HTTP layer (Wails WebView strips
// multipart Content-Type headers, breaking FormData uploads).
func (a *App) ImportBatchZip(filePath string) (*api.BatchImportResult, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read ZIP failed: %w", err)
	}
	return api.ProcessBatchZip(data, a.manager, a.registry)
}

// DeleteTemplate uninstalls a template by name.
// Bypasses the HTTP layer where Wails WebView may not support DELETE method.
func (a *App) DeleteTemplate(name string) error {
	return a.manager.Uninstall(name)
}

// InstallTemplate installs a template by name from the registry.
// Bypasses the HTTP layer where Wails WebView decodes %2F in URLs,
// breaking route matching for owner/repo path parameters.
func (a *App) InstallTemplate(templateName string) error {
	if a.registry == nil {
		return fmt.Errorf("registry not available")
	}

	entry := a.registry.LookupByName(templateName)
	if entry == nil {
		return fmt.Errorf("template %q not found in registry", templateName)
	}

	// Extract owner/repo
	parts := strings.SplitN(entry.Repo, "/", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid repo format: %s", entry.Repo)
	}
	owner, repo := parts[0], parts[1]

	// Build install opts from registry entry
	platform := runtime.GOOS + "-" + runtime.GOARCH
	var opts *template.InstallOpts
	if info, ok := entry.Platforms[platform]; ok && info.URL != "" {
		opts = &template.InstallOpts{
			DownloadURL:    info.URL,
			CdnURL:         info.CdnURL,
			ExpectedSHA256: info.SHA256,
			Trust:          entry.Trust,
			OnProgress: func(downloaded, total int64) {
				// Emit progress event to frontend
				if total > 0 {
					percent := float64(downloaded) / float64(total) * 100
					wailsRuntime.EventsEmit(a.ctx, "template-download:progress", map[string]interface{}{
						"template":   templateName,
						"downloaded": downloaded,
						"total":      total,
						"percent":    percent,
					})
				}
			},
		}
		logger.Info("[templates] Wails install", "name", templateName, "trust", entry.Trust, "platform", platform)
	}

	err := a.manager.Install(owner, repo, opts)
	if err != nil {
		return err
	}

	// Emit templates-changed event to refresh frontend
	wailsRuntime.EventsEmit(a.ctx, "templates-changed")
	return nil
}

// GetInstalledTemplates returns list of installed templates for frontend refresh
func (a *App) GetInstalledTemplates() ([]string, error) {
	templates, err := a.manager.List()
	if err != nil {
		return nil, err
	}

	names := make([]string, len(templates))
	for i, t := range templates {
		names[i] = t.Manifest.Name
	}
	return names, nil
}

// GetVersion returns the current app version.
func (a *App) GetVersion() string {
	if version == "" {
		return "dev"
	}
	return version
}

// UpdateInfo holds the result of a version check against GitHub releases.
type UpdateInfo struct {
	HasUpdate      bool   `json:"hasUpdate"`
	CurrentVersion string `json:"currentVersion"`
	LatestVersion  string `json:"latestVersion"`
	DownloadURL    string `json:"downloadURL"`
	ReleaseURL     string `json:"releaseURL"`
}

// CheckForUpdate queries GitHub for the latest release and compares with the current version.
func (a *App) CheckForUpdate() (*UpdateInfo, error) {
	current := a.GetVersion()
	info := &UpdateInfo{CurrentVersion: current}

	// NEW-02: Use HTTP client with timeout for update checks
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get("https://api.github.com/repos/Presto-io/Presto-Homepage/releases/latest")
	if err != nil {
		return nil, fmt.Errorf("failed to check update: %w", err)
	}
	defer resp.Body.Close()

	var release struct {
		TagName string `json:"tag_name"`
		HTMLURL string `json:"html_url"`
		Assets  []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to parse release: %w", err)
	}

	latest := strings.TrimPrefix(release.TagName, "v")
	info.LatestVersion = latest
	info.ReleaseURL = release.HTMLURL

	if latest != current && current != "dev" {
		info.HasUpdate = true
	}

	// Find a matching asset for the current platform
	osName := runtime.GOOS
	if osName == "darwin" {
		osName = "macOS"
	}
	pattern := fmt.Sprintf("%s-%s", osName, runtime.GOARCH)
	for _, asset := range release.Assets {
		if strings.Contains(asset.Name, pattern) {
			info.DownloadURL = asset.BrowserDownloadURL
			break
		}
	}

	return info, nil
}

// SaveFile decodes base64 data and saves it via native save dialog.
// Used by batch conversion page where blobs are created client-side.
func (a *App) SaveFile(b64Data string, defaultFilename string) error {
	data, err := base64.StdEncoding.DecodeString(b64Data)
	if err != nil {
		return fmt.Errorf("invalid data: %w", err)
	}

	savePath, err := wailsRuntime.SaveFileDialog(a.ctx, wailsRuntime.SaveDialogOptions{
		DefaultFilename: defaultFilename,
	})
	if err != nil {
		return fmt.Errorf("save dialog failed: %w", err)
	}
	if savePath == "" {
		return nil
	}

	if err := os.WriteFile(savePath, data, 0644); err != nil {
		return fmt.Errorf("write failed: %w", err)
	}
	log.Printf("[desktop] saved file %s (%d bytes)", defaultFilename, len(data))
	return nil
}

// SavePDF converts markdown to PDF and opens a native save dialog.
func (a *App) SavePDF(markdown string, templateId string, workDir string) error {
	tpl, err := a.manager.Get(templateId)
	if err != nil {
		return fmt.Errorf("template not found: %w", err)
	}

	executor := a.manager.Executor(tpl)
	typstOutput, err := executor.Convert(markdown)
	if err != nil {
		return fmt.Errorf("conversion failed: %w", err)
	}

	pdf, err := a.compiler.CompileString(typstOutput, workDir)
	if err != nil {
		return fmt.Errorf("compile failed: %w", err)
	}

	filename := extractTypstTitle(typstOutput) + ".pdf"

	savePath, err := wailsRuntime.SaveFileDialog(a.ctx, wailsRuntime.SaveDialogOptions{
		DefaultFilename: filename,
		Filters: []wailsRuntime.FileFilter{
			{DisplayName: "PDF Files", Pattern: "*.pdf"},
		},
	})
	if err != nil {
		return fmt.Errorf("save dialog failed: %w", err)
	}
	if savePath == "" {
		return nil // user cancelled
	}

	if err := os.WriteFile(savePath, pdf, 0644); err != nil {
		return fmt.Errorf("write failed: %w", err)
	}

	logger.Info("[desktop] saved PDF", "path", savePath, "bytes", len(pdf))
 return nil
}

// extractTypstTitle finds the first heading from typst source.
// Tries level 1 (=), then level 2 (==), etc. Falls back to "output".
func extractTypstTitle(typ string) string {
	lines := strings.Split(typ, "\n")
	// Try heading levels 1 through 5
	for level := 1; level <= 5; level++ {
		prefix := strings.Repeat("=", level) + " "
		deeperPrefix := strings.Repeat("=", level+1)
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if !strings.HasPrefix(trimmed, prefix) {
				continue
			}
			// Skip deeper headings (e.g. "== " when looking for "= ")
			if level < 5 && strings.HasPrefix(trimmed, deeperPrefix) {
				continue
			}
			content := strings.TrimSpace(trimmed[len(prefix):])
			title := resolveTypstText(content, lines)
			title = sanitizeFilename(title)
			if title != "" {
				return title
			}
		}
	}
	return "output"
}

// resolveTypstText extracts plain text from a typst heading content.
// If it's a variable reference like #varName..., resolves via #let varName = "value".
var letPattern = regexp.MustCompile(`#let\s+(\w+)\s*=\s*"([^"]*)"`)

func resolveTypstText(content string, lines []string) string {
	// Plain text heading (no typst expression)
	if !strings.HasPrefix(content, "#") {
		return content
	}
	// Extract variable name from expressions like #autoTitle.split(...) or #autoTitle
	varName := content[1:] // strip leading #
	if idx := strings.IndexAny(varName, ".( "); idx > 0 {
		varName = varName[:idx]
	}
	// Look for #let varName = "value"
	for _, line := range lines {
		m := letPattern.FindStringSubmatch(line)
		if m != nil && m[1] == varName {
			return m[2]
		}
	}
	return ""
}

func sanitizeFilename(s string) string {
	return strings.Map(func(r rune) rune {
		if strings.ContainsRune(`/\:*?"<>|`, r) {
			return '_'
		}
		return r
	}, strings.TrimSpace(s))
}

// findTypstBinary locates the typst binary.
// Search order: bundled in .app/Contents/Resources → next to executable → system PATH.
func findTypstBinary() string {
	exe, err := os.Executable()
	if err == nil {
		exe, _ = filepath.EvalSymlinks(exe)
		exeDir := filepath.Dir(exe)

		// macOS .app: Contents/MacOS/Presto → Contents/Resources/typst
		resources := filepath.Join(exeDir, "..", "Resources", "typst")
		if _, err := os.Stat(resources); err == nil {
			return resources
		}

		// Same directory as executable
		beside := filepath.Join(exeDir, "typst")
		if _, err := os.Stat(beside); err == nil {
			return beside
		}
	}

	// Fallback to system PATH
	if p, err := exec.LookPath("typst"); err == nil {
		return p
	}

	return "typst" // will fail at runtime with a clear error
}

// downloadTemplatesAndExit downloads all official templates and exits.
// Exit code 0 = success, 1 = failure.
// Used by NSIS installer for template pre-download during installation.
func initLogger() {
	// Determine log level
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}

	// Create multi-writer if log file is specified
	var writer io.Writer = os.Stderr

	if logFilePath != "" {
		// Use lumberjack for log rotation (10MB max, keep 5 backups)
		loggerLogFile = &lumberjack.Logger{
			Filename:   logFilePath,
			MaxSize:    10,   // megabytes
			MaxBackups: 5,    // keep at most 5 old log files
			MaxAge:     0,    // days (0 = don't delete based on age)
			Compress:   false, // don't compress old files
			LocalTime:  true,  // use local time for rotation timestamp
		}

		// Write to both stderr and file
		writer = io.MultiWriter(os.Stderr, loggerLogFile)

		logger.Info("[logger] log rotation enabled",
			"max_size_mb", 10,
			"max_backups", 5,
			"log_file", logFilePath)
	}

	// Create text handler (human-readable)
	opts := &slog.HandlerOptions{
		Level:       level,
		ReplaceAttr: sanitizeLogAttributes,
	}
	handler := slog.NewTextHandler(writer, opts)

	// Set global logger
	logger = slog.New(handler)
	slog.SetDefault(logger)

	logger.Info("[presto] logger initialized",
		"verbose", verbose,
		"log_file", logFilePath,
		"level", level.String())
}

// closeLogger closes the log file if opened
func closeLogger() {
	if loggerLogFile != nil {
		logger.Info("[presto] shutting down logger")
		loggerLogFile.Close()
	}
}

// sanitizeLogAttributes removes sensitive information from log attributes
func sanitizeLogAttributes(groups []string, a slog.Attr) slog.Attr {
	// Sanitize file paths (replace home directory)
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

// homeDir returns the user's home directory for sanitization
func homeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "" // Fallback: no sanitization
	}
	return home
}

func downloadTemplatesAndExit() {
	// Initialize logger for CLI mode (output to stdout for NSIS installer)
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	logger = slog.New(handler)
	slog.SetDefault(logger)

	logger.Info("[Installer] Starting template download...")

	home, err := os.UserHomeDir()
	if err != nil {
		logger.Error("[Installer] Failed to get user home directory", "error", err)
		os.Exit(1)
	}

	prestoDir := filepath.Join(home, ".presto")
	templatesDir := filepath.Join(prestoDir, "templates")
	if err := os.MkdirAll(templatesDir, 0755); err != nil {
		logger.Error("[Installer] Failed to create templates directory", "error", err)
		os.Exit(1)
	}

	manager := template.NewManager(templatesDir)
	registry := template.NewRegistryCache(prestoDir)

	// Load registry (fetches from CDN if cache is missing/expired)
	reg := registry.Load()
	if reg == nil {
		logger.Error("[Installer] Failed to load template registry")
		os.Exit(1)
	}

	// Filter official templates
	var officialTemplates []template.RegistryEntry
	for _, entry := range reg.Templates {
		if entry.Trust == "official" {
			officialTemplates = append(officialTemplates, entry)
		}
	}

	if len(officialTemplates) == 0 {
		logger.Info("[Installer] No official templates found in registry")
		os.Exit(0)
	}

	logger.Info("[Installer] Found official templates to download", "count", len(officialTemplates))

	platform := runtime.GOOS + "-" + runtime.GOARCH
	var failCount int
	for _, entry := range officialTemplates {
		// Skip if already installed
		if manager.Exists(entry.Name) {
			logger.Info("[Installer] Template already installed, skipping", "name", entry.Name)
			continue
		}

		parts := strings.SplitN(entry.Repo, "/", 2)
		if len(parts) != 2 {
			logger.Error("[Installer] Invalid repo format", "name", entry.Name, "repo", entry.Repo)
			failCount++
			continue
		}
		owner, repo := parts[0], parts[1]

		var opts *template.InstallOpts
		if info, ok := entry.Platforms[platform]; ok && info.URL != "" {
			opts = &template.InstallOpts{
				DownloadURL:    info.URL,
				CdnURL:         info.CdnURL,
				ExpectedSHA256: info.SHA256,
				Trust:          entry.Trust,
			}
		}

		logger.Info("[Installer] Downloading template", "name", entry.Name)
		if err := manager.Install(owner, repo, opts); err != nil {
			logger.Error("[Installer] Failed to download template", "name", entry.Name, "error", err)
			failCount++
			continue
		}
		logger.Info("[Installer] Successfully downloaded template", "name", entry.Name)
	}

	if failCount > 0 {
		logger.Error("[Installer] Template download completed with failures", "fail_count", failCount)
		os.Exit(1)
	}

	logger.Info("[Installer] Template download completed successfully")
	os.Exit(0)
}

func main() {
	// Parse command-line flags
	flag.Parse()

	// Initialize logger
	initLogger()

	// Check for --download-templates flag (used by NSIS installer)
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

	// Registry cache for SHA256 verification of imported templates
	registry := template.NewRegistryCache(prestoDir)
	registry.RefreshAsync()

	// SEC-40: Use os temp dir instead of $HOME to restrict file access
	compiler := typst.NewCompilerWithRoot(os.TempDir())
	compiler.BinPath = typstBin

	// Reuse existing API server as HTTP handler for /api/* routes
	apiHandler := api.NewServer(api.ServerOptions{
		TemplatesDir: templatesDir,
		TypstBin:     typstBin,
		Registry:     registry,
	})

	// Strip "build" prefix from embedded FS so files are at root
	frontendFS, _ := fs.Sub(assets, "build")

	// Wrap API handler with SPA fallback for prerendered routes
	// Wails calls Handler when the asset is not found in the embedded FS.
	// For prerendered SvelteKit routes like /showcase/editor, the actual
	// file is showcase/editor.html — try .html suffix before API fallback.
	handler := &spaFallbackHandler{api: apiHandler, assets: frontendFS}

	app := NewApp(manager, compiler, registry)
	appMenu := buildMenu(app)

	// Check os.Args for a presto:// URL (cold start via URL scheme)
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
		Bind: []interface{}{
			app,
		},
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
					// Hot start: frontend is ready, emit event directly
					go app.handlePrestoURL(url)
				} else {
					// Cold start: frontend not ready yet, store for later pull
					startupURL = url
				}
			},
		},
	})
	if err != nil {
		println("Error:", err.Error())
	}
}
