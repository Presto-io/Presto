package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/wailsapp/wails/v2"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	macOptions "github.com/wailsapp/wails/v2/pkg/options/mac"

	"github.com/mrered/presto/internal/api"
	"github.com/mrered/presto/internal/template"
	"github.com/mrered/presto/internal/typst"
)

//go:embed all:build
var assets embed.FS

type App struct {
	ctx      context.Context
	manager  *template.Manager
	compiler *typst.Compiler
}

type OpenFileResult struct {
	Content string `json:"content"`
	Dir     string `json:"dir"`
}

func NewApp(manager *template.Manager, compiler *typst.Compiler) *App {
	return &App{manager: manager, compiler: compiler}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
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

func buildMenu(app *App) *menu.Menu {
	appMenu := menu.NewMenu()
	appMenu.Append(menu.AppMenu())

	fileMenu := appMenu.AddSubmenu("文件")
	fileMenu.AddText("打开 Markdown…", keys.CmdOrCtrl("o"), func(_ *menu.CallbackData) {
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

	appMenu.Append(menu.EditMenu())
	appMenu.Append(menu.WindowMenu())
	return appMenu
}

// SavePDF converts markdown to PDF and opens a native save dialog.
func (a *App) SavePDF(markdown string, templateId string, workDir string) error {
	tpl, err := a.manager.Get(templateId)
	if err != nil {
		return fmt.Errorf("template not found: %w", err)
	}

	exec := a.manager.Executor(tpl)
	typstOutput, err := exec.Convert(markdown)
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
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if !strings.HasPrefix(trimmed, prefix) {
				continue
			}
			// Make sure it's exactly this level, not a deeper one
			// e.g. "= " is level 1, "== " is level 2
			if level < 5 {
				deeper := strings.Repeat("=", level+1)
				if strings.HasPrefix(trimmed, deeper) {
					continue
				}
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
	home, _ := os.UserHomeDir()
	templatesDir := filepath.Join(home, ".presto", "templates")
	os.MkdirAll(templatesDir, 0755)

	manager := template.NewManager(templatesDir)
	typstBin := findTypstBinary()
	log.Printf("[presto] using typst: %s", typstBin)

	compiler := typst.NewCompilerWithRoot("/")
	compiler.BinPath = typstBin

	// Reuse existing API server as HTTP handler for /api/* routes
	apiHandler := api.NewServer(templatesDir, "", typstBin)

	// Strip "build" prefix from embedded FS so files are at root
	frontendFS, _ := fs.Sub(assets, "build")

	app := NewApp(manager, compiler)
	appMenu := buildMenu(app)

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
		Menu:      appMenu,
		OnStartup: app.startup,
		Bind: []interface{}{
			app,
		},
		Mac: &macOptions.Options{
			TitleBar: macOptions.TitleBarHiddenInset(),
			About: &macOptions.AboutInfo{
				Title:   "Presto",
				Message: "Markdown → Typst → PDF",
			},
		},
	})
	if err != nil {
		println("Error:", err.Error())
	}
}
