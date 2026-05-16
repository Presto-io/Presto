package main

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	updateReleaseAPIURL      = "https://api.github.com/repos/Presto-io/Presto-Homepage/releases/latest"
	updateChecksumsAssetName = "checksums.txt"
	maxUpdateAPIBodyBytes    = 2 * 1024 * 1024
	maxUpdateChecksumBytes   = 1024 * 1024
	maxUpdateDownloadBytes   = 500 * 1024 * 1024
	maxUpdateExtractedBytes  = 1024 * 1024 * 1024
	maxUpdateExtractedFiles  = 2000
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
	resp, err := client.Get(updateReleaseAPIURL)
	if err != nil {
		return nil, fmt.Errorf("failed to check update: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to check update: HTTP %d", resp.StatusCode)
	}

	var release struct {
		TagName string `json:"tag_name"`
		HTMLURL string `json:"html_url"`
		Assets  []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxUpdateAPIBodyBytes)).Decode(&release); err != nil {
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
		if strings.Contains(asset.Name, pattern) && isExpectedUpdateAsset(asset.Name) {
			if _, _, _, err := parseUpdateAssetURL(asset.BrowserDownloadURL); err != nil {
				log.Printf("[desktop] skipped unsafe update asset URL for %s: %v", asset.Name, err)
				continue
			}
			info.DownloadURL = asset.BrowserDownloadURL
			break
		}
	}

	return info, nil
}

// CheckStartupUpdate checks for updates at launch and prompts only when an update exists.
func (a *App) CheckStartupUpdate() {
	info, err := a.CheckForUpdate()
	if err != nil {
		log.Printf("[desktop] startup update check failed: %v", err)
		return
	}
	if info.HasUpdate {
		a.promptForUpdate(info)
	}
}

// CheckAndNotifyUpdate checks for updates and shows a dialog for the result.
func (a *App) CheckAndNotifyUpdate() {
	info, err := a.CheckForUpdate()
	if err != nil {
		log.Printf("[desktop] update check failed: %v", err)
		if a.ctx != nil {
			wailsRuntime.MessageDialog(a.ctx, wailsRuntime.MessageDialogOptions{
				Type:    wailsRuntime.ErrorDialog,
				Title:   "检查更新",
				Message: fmt.Sprintf("检查更新失败：%v", err),
			})
		}
		return
	}
	if info.HasUpdate {
		a.promptForUpdate(info)
	} else {
		wailsRuntime.MessageDialog(a.ctx, wailsRuntime.MessageDialogOptions{
			Type:    wailsRuntime.InfoDialog,
			Title:   "检查更新",
			Message: "已是最新版本",
		})
	}
}

func (a *App) promptForUpdate(info *UpdateInfo) {
	if a.ctx == nil || info == nil || !info.HasUpdate {
		return
	}

	action := "下载并安装"
	if info.DownloadURL == "" {
		action = "查看发布页面"
	}
	result, err := wailsRuntime.MessageDialog(a.ctx, wailsRuntime.MessageDialogOptions{
		Type:          wailsRuntime.InfoDialog,
		Title:         "发现新版本",
		Message:       fmt.Sprintf("Presto %s 已可用。\n当前版本：%s\n\n现在更新吗？", info.LatestVersion, info.CurrentVersion),
		Buttons:       []string{action, "稍后"},
		DefaultButton: "稍后",
		CancelButton:  "稍后",
	})
	if err != nil {
		log.Printf("[desktop] update prompt failed: %v", err)
		return
	}
	if result != action {
		return
	}
	if info.DownloadURL == "" {
		if info.ReleaseURL != "" {
			wailsRuntime.BrowserOpenURL(a.ctx, info.ReleaseURL)
		}
		return
	}
	go func() {
		if err := a.DownloadAndInstallUpdate(info.DownloadURL); err != nil {
			log.Printf("[desktop] update install failed: %v", err)
			if a.ctx != nil {
				wailsRuntime.MessageDialog(a.ctx, wailsRuntime.MessageDialogOptions{
					Type:    wailsRuntime.ErrorDialog,
					Title:   "更新失败",
					Message: fmt.Sprintf("更新失败：%v", err),
				})
			}
		}
	}()
}

// DownloadAndInstallUpdate downloads the release asset and installs it in-place.
// Progress is reported via Wails events: "update:progress" (int 0-100), "update:status" (string).
func (a *App) DownloadAndInstallUpdate(downloadURL string) error {
	if downloadURL == "" {
		return fmt.Errorf("no download URL")
	}
	repo, tag, filename, err := parseUpdateAssetURL(downloadURL)
	if err != nil {
		return fmt.Errorf("unsafe update URL: %w", err)
	}
	if !isExpectedUpdateAsset(filename) {
		return fmt.Errorf("unexpected update asset for this platform: %s", filename)
	}

	wailsRuntime.EventsEmit(a.ctx, "update:status", "正在下载更新…")
	wailsRuntime.EventsEmit(a.ctx, "update:progress", 0)

	client := &http.Client{Timeout: 10 * time.Minute}
	expectedHash, err := fetchUpdateChecksum(client, repo, tag, filename)
	if err != nil {
		return err
	}

	resp, err := client.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}
	if resp.ContentLength > maxUpdateDownloadBytes {
		return fmt.Errorf("download too large: %d bytes", resp.ContentLength)
	}

	tmpDir, err := os.MkdirTemp("", "presto-update-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}

	tmpFile := filepath.Join(tmpDir, filename)

	if err := a.downloadToFile(resp, tmpFile, expectedHash); err != nil {
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

func parseUpdateAssetURL(rawURL string) (repo, tag, filename string, err error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", "", "", err
	}
	if u.Scheme != "https" || !strings.EqualFold(u.Host, "github.com") {
		return "", "", "", fmt.Errorf("host must be github.com over HTTPS")
	}
	parts := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
	if len(parts) != 6 || parts[0] != "Presto-io" || parts[2] != "releases" || parts[3] != "download" {
		return "", "", "", fmt.Errorf("URL must point to a Presto-io GitHub release asset")
	}
	if parts[1] != "Presto-Homepage" && parts[1] != "Presto" {
		return "", "", "", fmt.Errorf("repo is not allowed")
	}
	if parts[4] == "" || parts[5] == "" || parts[5] == "." || parts[5] == ".." || strings.Contains(parts[5], string(filepath.Separator)) {
		return "", "", "", fmt.Errorf("release tag or asset name is empty")
	}
	if filepath.Base(parts[5]) != parts[5] {
		return "", "", "", fmt.Errorf("asset name must be a basename")
	}
	return parts[1], parts[4], parts[5], nil
}

func isExpectedUpdateAsset(filename string) bool {
	osName := runtime.GOOS
	if osName == "darwin" {
		osName = "macOS"
	}
	platform := fmt.Sprintf("-%s-%s", osName, runtime.GOARCH)
	if !strings.HasPrefix(filename, "Presto-") || !strings.Contains(filename, platform) {
		return false
	}
	switch runtime.GOOS {
	case "darwin":
		return strings.HasSuffix(filename, ".dmg")
	case "windows":
		return strings.HasSuffix(filename, "-installer.exe") || strings.HasSuffix(filename, ".zip")
	case "linux":
		return strings.HasSuffix(filename, ".tar.gz")
	default:
		return false
	}
}

func fetchUpdateChecksum(client *http.Client, repo, tag, filename string) (string, error) {
	checksumURL := fmt.Sprintf("https://github.com/Presto-io/%s/releases/download/%s/%s", repo, url.PathEscape(tag), updateChecksumsAssetName)
	resp, err := client.Get(checksumURL)
	if err != nil {
		return "", fmt.Errorf("failed to download update checksums: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download update checksums: HTTP %d", resp.StatusCode)
	}
	data, err := readLimited(resp.Body, maxUpdateChecksumBytes)
	if err != nil {
		return "", fmt.Errorf("failed to read update checksums: %w", err)
	}
	checksums := parseUpdateChecksums(data)
	expected := checksums[filename]
	if expected == "" {
		return "", fmt.Errorf("missing SHA256 checksum for update asset: %s", filename)
	}
	return expected, nil
}

func parseUpdateChecksums(data []byte) map[string]string {
	checksums := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 || !isHexSHA256(fields[0]) {
			continue
		}
		name := strings.TrimPrefix(fields[1], "*")
		name = filepath.Base(name)
		checksums[name] = strings.ToLower(fields[0])
	}
	return checksums
}

func isHexSHA256(s string) bool {
	if len(s) != 64 {
		return false
	}
	for _, r := range s {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			return false
		}
	}
	return true
}

func readLimited(r io.Reader, maxBytes int64) ([]byte, error) {
	lr := &io.LimitedReader{R: r, N: maxBytes + 1}
	data, err := io.ReadAll(lr)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("data exceeds %d bytes", maxBytes)
	}
	return data, nil
}

func (a *App) downloadToFile(resp *http.Response, dst string, expectedHash string) error {
	f, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	total := resp.ContentLength
	var downloaded int64
	hasher := sha256.New()
	buf := make([]byte, 32*1024)
	lastPct := -1
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			downloaded += int64(n)
			if downloaded > maxUpdateDownloadBytes {
				return fmt.Errorf("download exceeds %d bytes", maxUpdateDownloadBytes)
			}
			if _, wErr := f.Write(buf[:n]); wErr != nil {
				return fmt.Errorf("write failed: %w", wErr)
			}
			if _, hErr := hasher.Write(buf[:n]); hErr != nil {
				return fmt.Errorf("checksum failed: %w", hErr)
			}
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
	actualHash := fmt.Sprintf("%x", hasher.Sum(nil))
	if !strings.EqualFold(actualHash, expectedHash) {
		return fmt.Errorf("update checksum mismatch")
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

// --- Windows: launch installer or extract legacy zip, then relaunch/quit ---

func (a *App) installWindows(archivePath, tmpDir string) error {
	switch strings.ToLower(filepath.Ext(archivePath)) {
	case ".exe":
		return a.installWindowsInstaller(archivePath, tmpDir)
	case ".zip":
		return a.installWindowsZip(archivePath, tmpDir)
	default:
		return fmt.Errorf("unsupported Windows update package: %s", filepath.Base(archivePath))
	}
}

func (a *App) installWindowsInstaller(installerPath, tmpDir string) error {
	// Launch the installer shortly after the app exits to avoid file locking issues.
	script := fmt.Sprintf("@echo off\r\ntimeout /t 2 /nobreak >nul\r\nstart \"\" \"%s\"\r\n", installerPath)
	scriptPath := filepath.Join(tmpDir, "launch-installer.bat")
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		return fmt.Errorf("write installer launcher: %w", err)
	}

	wailsRuntime.EventsEmit(a.ctx, "update:status", "正在启动安装程序…")
	execCommand("cmd", "/c", "start", "/b", scriptPath).Start()
	wailsRuntime.Quit(a.ctx)
	return nil
}

func (a *App) installWindowsZip(zipPath, tmpDir string) error {
	extractDir := filepath.Join(tmpDir, "extracted")
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		return fmt.Errorf("mkdir extract: %w", err)
	}

	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("open zip: %w", err)
	}
	defer r.Close()

	var extractedBytes int64
	fileCount := 0
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
		if !f.FileInfo().Mode().IsRegular() {
			return fmt.Errorf("unsupported zip entry type: %s", f.Name)
		}
		fileCount++
		if fileCount > maxUpdateExtractedFiles {
			return fmt.Errorf("zip contains too many files")
		}
		if int64(f.UncompressedSize64) > maxUpdateExtractedBytes-extractedBytes {
			return fmt.Errorf("zip extracted data exceeds %d bytes", maxUpdateExtractedBytes)
		}
		if err := extractZipFile(f, target, &extractedBytes); err != nil {
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
	execCommand("cmd", "/c", "start", "/b", scriptPath).Start()
	wailsRuntime.Quit(a.ctx)
	return nil
}

func extractZipFile(f *zip.File, target string, extractedBytes *int64) error {
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
	remaining := maxUpdateExtractedBytes - *extractedBytes
	lr := &io.LimitedReader{R: rc, N: remaining + 1}
	n, err := io.Copy(w, lr)
	*extractedBytes += n
	if n > remaining {
		return fmt.Errorf("zip extracted data exceeds %d bytes", maxUpdateExtractedBytes)
	}
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
	var extractedBytes int64
	fileCount := 0
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
			fileCount++
			if fileCount > maxUpdateExtractedFiles {
				return fmt.Errorf("tar contains too many files")
			}
			if hdr.Size < 0 || hdr.Size > maxUpdateExtractedBytes-extractedBytes {
				return fmt.Errorf("tar extracted data exceeds %d bytes", maxUpdateExtractedBytes)
			}
			if err := extractTarFile(tr, target, os.FileMode(hdr.Mode), hdr.Size, &extractedBytes); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported tar entry type for update package: %s", hdr.Name)
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
		if e.IsDir() {
			continue
		}
		src := filepath.Join(extractDir, e.Name())
		dst := filepath.Join(exeDir, e.Name())
		info, err := os.Stat(src)
		if err != nil {
			continue
		}
		if err := copyFile(dst, src, info.Mode()); err != nil {
			return fmt.Errorf("copy update file: %w", err)
		}
	}

	os.RemoveAll(tmpDir)

	wailsRuntime.EventsEmit(a.ctx, "update:status", "更新完成，正在重启…")
	cmd := exec.Command(exePath)
	cmd.Start()
	wailsRuntime.Quit(a.ctx)
	return nil
}

func extractTarFile(tr *tar.Reader, target string, mode os.FileMode, size int64, extractedBytes *int64) error {
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return err
	}
	w, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer w.Close()
	n, err := io.CopyN(w, tr, size)
	*extractedBytes += n
	if err != nil {
		return err
	}
	if n != size {
		return fmt.Errorf("short tar entry: wrote %d of %d bytes", n, size)
	}
	return nil
}

func copyFile(dst, src string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
