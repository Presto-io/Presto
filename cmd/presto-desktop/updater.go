package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

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

// DownloadAndInstallUpdate downloads the release asset and installs it in-place.
// Progress is reported via Wails events: "update:progress" (int 0-100), "update:status" (string).
func (a *App) DownloadAndInstallUpdate(downloadURL string) error {
	if downloadURL == "" {
		return fmt.Errorf("no download URL")
	}

	wailsRuntime.EventsEmit(a.ctx, "update:status", "正在下载更新…")
	wailsRuntime.EventsEmit(a.ctx, "update:progress", 0)

	client := &http.Client{Timeout: 10 * time.Minute}
	resp, err := client.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}

	tmpDir, err := os.MkdirTemp("", "presto-update-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}

	parts := strings.Split(downloadURL, "/")
	filename := parts[len(parts)-1]
	tmpFile := filepath.Join(tmpDir, filename)

	if err := a.downloadToFile(resp, tmpFile); err != nil {
		os.RemoveAll(tmpDir)
		return err
	}

	wailsRuntime.EventsEmit(a.ctx, "update:status", "正在安装更新…")

	switch runtime.GOOS {
	case "darwin":
		return a.installMacOS(tmpFile, tmpDir)
	case "windows":
		return a.installWindows(tmpFile, tmpDir)
	case "linux":
		return a.installLinux(tmpFile, tmpDir)
	default:
		os.RemoveAll(tmpDir)
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

func (a *App) downloadToFile(resp *http.Response, dst string) error {
	f, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	total := resp.ContentLength
	var downloaded int64
	buf := make([]byte, 32*1024)
	lastPct := -1
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, wErr := f.Write(buf[:n]); wErr != nil {
				return fmt.Errorf("write failed: %w", wErr)
			}
			downloaded += int64(n)
			if total > 0 {
				pct := int(float64(downloaded) / float64(total) * 100)
				if pct != lastPct {
					wailsRuntime.EventsEmit(a.ctx, "update:progress", pct)
					lastPct = pct
				}
			}
		}
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			return fmt.Errorf("download error: %w", readErr)
		}
	}
	wailsRuntime.EventsEmit(a.ctx, "update:progress", 100)
	return nil
}

// --- macOS: mount DMG, copy .app, relaunch ---

func (a *App) installMacOS(dmgPath, tmpDir string) error {
	mountPoint := filepath.Join(tmpDir, "mount")
	if err := os.MkdirAll(mountPoint, 0755); err != nil {
		return fmt.Errorf("mkdir mount: %w", err)
	}

	out, err := exec.Command("hdiutil", "attach", dmgPath, "-nobrowse", "-mountpoint", mountPoint).CombinedOutput()
	if err != nil {
		return fmt.Errorf("mount DMG failed: %s: %w", string(out), err)
	}
	defer exec.Command("hdiutil", "detach", mountPoint, "-quiet").Run()

	entries, err := os.ReadDir(mountPoint)
	if err != nil {
		return fmt.Errorf("read mount: %w", err)
	}
	var appName string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".app") {
			appName = e.Name()
			break
		}
	}
	if appName == "" {
		return fmt.Errorf("no .app found in DMG")
	}

	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get executable: %w", err)
	}
	// /Applications/Presto.app/Contents/MacOS/Presto → /Applications/Presto.app
	appBundle := filepath.Dir(filepath.Dir(filepath.Dir(exePath)))
	srcApp := filepath.Join(mountPoint, appName)

	// Replace: rm old, cp new (safe on macOS — running binary stays in memory)
	if out, err := exec.Command("rm", "-rf", appBundle).CombinedOutput(); err != nil {
		return fmt.Errorf("remove old app: %s: %w", string(out), err)
	}
	if out, err := exec.Command("cp", "-R", srcApp, appBundle).CombinedOutput(); err != nil {
		return fmt.Errorf("copy new app: %s: %w", string(out), err)
	}

	wailsRuntime.EventsEmit(a.ctx, "update:status", "更新完成，正在重启…")
	exec.Command("open", "-n", appBundle).Start()
	wailsRuntime.Quit(a.ctx)
	return nil
}

// --- Windows: extract zip, schedule replacement via script, relaunch ---

func (a *App) installWindows(zipPath, tmpDir string) error {
	extractDir := filepath.Join(tmpDir, "extracted")
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		return fmt.Errorf("mkdir extract: %w", err)
	}

	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("open zip: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		target := filepath.Join(extractDir, f.Name)
		// SEC: prevent zip slip
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(extractDir)+string(os.PathSeparator)) {
			continue
		}
		if f.FileInfo().IsDir() {
			os.MkdirAll(target, f.Mode())
			continue
		}
		if err := extractZipFile(f, target); err != nil {
			return err
		}
	}

	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get executable: %w", err)
	}
	exeDir := filepath.Dir(exePath)

	// Write a batch script that waits, copies, and relaunches
	script := fmt.Sprintf("@echo off\r\ntimeout /t 2 /nobreak >nul\r\nxcopy /y /e \"%s\\*\" \"%s\\\"\r\nstart \"\" \"%s\"\r\nrmdir /s /q \"%s\"\r\n",
		extractDir, exeDir, exePath, tmpDir)
	scriptPath := filepath.Join(tmpDir, "update.bat")
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		return fmt.Errorf("write script: %w", err)
	}

	wailsRuntime.EventsEmit(a.ctx, "update:status", "更新完成，正在重启…")
	exec.Command("cmd", "/c", "start", "/b", scriptPath).Start()
	wailsRuntime.Quit(a.ctx)
	return nil
}

func extractZipFile(f *zip.File, target string) error {
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return err
	}
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	w, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
	if err != nil {
		return err
	}
	defer w.Close()
	_, err = io.Copy(w, rc)
	return err
}

// --- Linux: extract tar.gz, replace binary, relaunch ---

func (a *App) installLinux(tarPath, tmpDir string) error {
	extractDir := filepath.Join(tmpDir, "extracted")
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		return fmt.Errorf("mkdir extract: %w", err)
	}

	f, err := os.Open(tarPath)
	if err != nil {
		return fmt.Errorf("open tar: %w", err)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("gzip reader: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("tar read: %w", err)
		}
		target := filepath.Join(extractDir, hdr.Name)
		// SEC: prevent path traversal
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(extractDir)+string(os.PathSeparator)) {
			continue
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			os.MkdirAll(target, os.FileMode(hdr.Mode))
		case tar.TypeReg:
			if err := extractTarFile(tr, target, os.FileMode(hdr.Mode)); err != nil {
				return err
			}
		}
	}

	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get executable: %w", err)
	}
	exeDir := filepath.Dir(exePath)

	// Copy extracted files over current installation
	entries, _ := os.ReadDir(extractDir)
	for _, e := range entries {
		src := filepath.Join(extractDir, e.Name())
		dst := filepath.Join(exeDir, e.Name())
		data, err := os.ReadFile(src)
		if err != nil {
			continue
		}
		info, _ := os.Stat(src)
		os.WriteFile(dst, data, info.Mode())
	}

	os.RemoveAll(tmpDir)

	wailsRuntime.EventsEmit(a.ctx, "update:status", "更新完成，正在重启…")
	cmd := exec.Command(exePath)
	cmd.Start()
	wailsRuntime.Quit(a.ctx)
	return nil
}

func extractTarFile(tr *tar.Reader, target string, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return err
	}
	w, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer w.Close()
	_, err = io.Copy(w, tr)
	return err
}
