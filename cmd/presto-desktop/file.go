package main

import (
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type OpenFileResult struct {
	Content string `json:"content"`
	Dir     string `json:"dir"`
	Path    string `json:"path"`
}

type OpenFilesItem struct {
	Name    string `json:"name"`
	Content string `json:"content"`
	Dir     string `json:"dir"`
	IsZip   bool   `json:"isZip"`
	Path    string `json:"path,omitempty"`
}

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
		Path:    path,
	}, nil
}

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
			item.Path = p
		} else {
			data, err := os.ReadFile(p)
			if err != nil {
				log.Printf("[desktop] failed to read %s: %v", p, err)
				continue
			}
			item.Content = string(data)
			item.Path = p
		}
		items = append(items, item)
	}
	return items
}

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

func (a *App) SaveMarkdown(content string, filePath string) error {
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("save failed: %w", err)
	}
	logger.Info("[desktop] saved markdown", "path", filePath, "bytes", len(content))
	return nil
}

func (a *App) SaveMarkdownAs(content string, defaultFilename string) (string, error) {
	if defaultFilename == "" {
		defaultFilename = "untitled.md"
	}
	if !strings.HasSuffix(strings.ToLower(defaultFilename), ".md") {
		defaultFilename += ".md"
	}
	savePath, err := wailsRuntime.SaveFileDialog(a.ctx, wailsRuntime.SaveDialogOptions{
		DefaultFilename: defaultFilename,
		Filters: []wailsRuntime.FileFilter{
			{DisplayName: "Markdown", Pattern: "*.md"},
			{DisplayName: "所有文件", Pattern: "*.*"},
		},
	})
	if err != nil {
		return "", fmt.Errorf("save dialog failed: %w", err)
	}
	if savePath == "" {
		return "", nil
	}
	if err := os.WriteFile(savePath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("write failed: %w", err)
	}
	logger.Info("[desktop] saved markdown as", "path", savePath, "bytes", len(content))
	return savePath, nil
}

func (a *App) ConfirmSaveDialog(filename string) (string, error) {
	msg := "是否保存对文档的更改？"
	if filename != "" {
		msg = fmt.Sprintf("是否保存对 \"%s\" 的更改？", filename)
	}
	result, err := wailsRuntime.MessageDialog(a.ctx, wailsRuntime.MessageDialogOptions{
		Type:          wailsRuntime.WarningDialog,
		Title:         "Presto",
		Message:       msg,
		Buttons:       []string{"保存", "不保存", "取消"},
		DefaultButton: "保存",
		CancelButton:  "取消",
	})
	if err != nil {
		return "Cancel", err
	}
	switch result {
	case "保存":
		return "Save", nil
	case "不保存":
		return "Don't Save", nil
	default:
		return "Cancel", nil
	}
}

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
		return nil
	}

	if err := os.WriteFile(savePath, pdf, 0644); err != nil {
		return fmt.Errorf("write failed: %w", err)
	}

	logger.Info("[desktop] saved PDF", "path", savePath, "bytes", len(pdf))
	wailsRuntime.EventsEmit(a.ctx, "app:notification", map[string]string{
		"message": "PDF 导出成功",
		"type":    "success",
	})
	return nil
}
