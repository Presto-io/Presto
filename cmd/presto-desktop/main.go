package main

import (
	"context"
	"embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
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
)

// version is set at build time via -ldflags "-X main.version=..."
var version string

// startupURL holds a presto:// URL passed via os.Args on cold start.
var startupURL string

//go:embed all:build
var assets embed.FS

type App struct {
	ctx      context.Context
	manager  *template.Manager
	compiler *typst.Compiler
	registry *template.RegistryCache
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
	templates, err := a.manager.List()
	if err != nil {
		log.Printf("[first-launch] failed to list templates: %v", err)
		return
	}

	// First launch if no templates installed
	if len(templates) == 0 {
		log.Printf("[first-launch] first launch detected, starting default template download")
		a.downloadDefaultTemplates()
		return
	}

	// Templates already installed - check for updates
	log.Printf("[first-launch] %d templates already installed, checking for updates", len(templates))
	go a.checkTemplateUpdates(templates)
}

// downloadDefaultTemplates downloads all official templates concurrently.
// Emits events for frontend to track progress.
func (a *App) downloadDefaultTemplates() {
	reg := a.registry.Load()
	if reg == nil {
		log.Printf("[first-launch] registry not available, skipping default download")
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
		log.Printf("[first-launch] no official templates found")
		return
	}

	log.Printf("[first-launch] downloading %d official templates", len(officialTemplates))

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

			log.Printf("[first-launch] downloading: %s", templateName)
			err := a.InstallTemplate(templateName)

			mu.Lock()
			if err != nil {
				failureCount++
				log.Printf("[first-launch] failed to download %s: %v", templateName, err)
				a.emitFirstLaunchProgress(templateName, "error", err.Error())
			} else {
				successCount++
				log.Printf("[first-launch] successfully downloaded: %s", templateName)
				a.emitFirstLaunchProgress(templateName, "success", "")
			}
			mu.Unlock()
		}(name)
	}

	wg.Wait()

	// Emit completion event
	a.emitFirstLaunchComplete(successCount, failureCount)
	log.Printf("[first-launch] download complete: %d success, %d failed", successCount, failureCount)
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
	log.Printf("[template-update] starting silent update check for %d templates", len(installed))

	reg := a.registry.Load()
	if reg == nil {
		log.Printf("[template-update] registry not available, skipping update check")
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
			log.Printf("[template-update] update available for %s: installed=%s, latest=%s",
				inst.Manifest.Name, inst.Manifest.Version, entry.Version)
			updatesAvailable = append(updatesAvailable, struct {
				name    string
				latest  template.RegistryEntry
				current string
			}{inst.Manifest.Name, *entry, inst.Manifest.Version})
		}
	}

	if len(updatesAvailable) == 0 {
		log.Printf("[template-update] all templates are up to date")
		return
	}

	log.Printf("[template-update] silently updating %d templates in background...", len(updatesAvailable))

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

			log.Printf("[template-update] updating %s from %s to %s...", name, oldVersion, entry.Version)

			// Delete old version first
			if err := a.manager.Uninstall(name); err != nil {
				log.Printf("[template-update] failed to uninstall old version of %s: %v", name, err)
				mu.Lock()
				failCount++
				mu.Unlock()
				return
			}

			// Install new version (silent - no events, no notifications)
			if err := a.InstallTemplate(name); err != nil {
				log.Printf("[template-update] failed to update %s: %v", name, err)
				mu.Lock()
				failCount++
				mu.Unlock()
				// NOTE: Old version already deleted, template is now missing
				// User will need to manually re-install (known limitation for v1)
				return
			}

			log.Printf("[template-update] successfully updated %s to version %s", name, entry.Version)
			mu.Lock()
			successCount++
			mu.Unlock()
		}(update.name, update.latest, update.current)
	}

	wg.Wait()

	// Log summary (for debugging)
	log.Printf("[template-update] silent update complete: %d success, %d failed", successCount, failCount)

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
	log.Printf("[url-scheme] GetStartupURL called, returning: %q", u)
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

	fileMenu := appMenu.AddSubmenu("文件")
	fileMenu.AddText("打开文件…", keys.CmdOrCtrl("o"), func(_ *menu.CallbackData) {
		wailsRuntime.EventsEmit(app.ctx, "menu:open")
	})
	fileMenu.AddSeparator()
	fileMenu.AddText("导出 PDF…", keys.CmdOrCtrl("e"), func(_ *menu.CallbackData) {
		wailsRuntime.EventsEmit(app.ctx, "menu:export")
	})
	fileMenu.AddSeparator()
	fileMenu.AddText("设置…", keys.CmdOrCtrl(","), func(_ *menu.CallbackData) {
		wailsRuntime.EventsEmit(app.ctx, "menu:settings")
	})
	fileMenu.AddSeparator()
	fileMenu.AddText("模板管理…", keys.Combo("t", keys.CmdOrCtrlKey, keys.ShiftKey), func(_ *menu.CallbackData) {
		wailsRuntime.EventsEmit(app.ctx, "menu:templates")
	})

	appMenu.Append(menu.EditMenu())

	windowMenu := appMenu.AddSubmenu("窗口")
	windowMenu.AddText("最小化", keys.CmdOrCtrl("m"), func(_ *menu.CallbackData) {
		wailsRuntime.WindowMinimise(app.ctx)
	})
	windowMenu.AddText("缩放", nil, func(_ *menu.CallbackData) {
		wailsRuntime.WindowToggleMaximise(app.ctx)
	})
	windowMenu.AddSeparator()
	windowMenu.AddText("关闭窗口", keys.CmdOrCtrl("w"), func(_ *menu.CallbackData) {
		wailsRuntime.Quit(app.ctx)
	})

	return appMenu
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
		log.Printf("[templates] Wails install: %s (trust=%s, platform=%s)", templateName, entry.Trust, platform)
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

	log.Printf("[desktop] saved PDF to %s (%d bytes)", savePath, len(pdf))
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

func main() {
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
	log.Printf("[presto] using typst: %s", typstBin)

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
	log.Printf("[url-scheme] os.Args: %v", os.Args)
	for _, arg := range os.Args[1:] {
		if strings.HasPrefix(arg, "presto://") {
			startupURL = arg
			log.Printf("[url-scheme] captured startup URL: %s", arg)
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
				log.Printf("[url-scheme] second instance args: %v", data.Args)
			},
		},
		Mac: &macOptions.Options{
			TitleBar: macOptions.TitleBarHiddenInset(),
			About: &macOptions.AboutInfo{
				Title:   "Presto",
				Message: "Markdown → Typst → PDF",
			},
			OnUrlOpen: func(url string) {
				log.Printf("[url-scheme] OnUrlOpen: %s", url)
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
