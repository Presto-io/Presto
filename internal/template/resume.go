package template

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	tmpDownloadDir = "presto-downloads"
)

// downloadWithResume downloads a file with resume support via HTTP Range requests.
// If a partial download exists, it resumes from the last byte.
// Returns downloaded data or error.
func downloadWithResume(downloadURL string, maxRetries int, onProgress ProgressCallback) ([]byte, error) {
	// Validate URL domain
	parsedURL, err := url.Parse(downloadURL)
	if err != nil {
		return nil, &InstallError{
			Type:    ErrNotFound,
			Message: fmt.Sprintf("invalid download URL: %v", err),
			Err:     err,
		}
	}
	if !isAllowedDownloadHost(parsedURL.Host) {
		slog.Warn("[security] blocked download URL host not in whitelist",
			"host", parsedURL.Host,
			"url", SanitizeURL(downloadURL))
		return nil, &InstallError{
			Type:    ErrNotFound,
			Message: fmt.Sprintf("download URL host not allowed: %s", parsedURL.Host),
			Err:     fmt.Errorf("host not allowed: %s", parsedURL.Host),
		}
	}

	// Create temp directory for partial downloads
	tmpDir := filepath.Join(os.TempDir(), tmpDownloadDir)
	if err := os.MkdirAll(tmpDir, 0700); err != nil {
		slog.Warn("[download] failed to create tmp dir, falling back to non-resumable download",
			"error", err.Error())
		// Fallback to non-resumable download
		return downloadWithRetry(downloadURL, maxRetries, onProgress)
	}

	// Generate temp file name based on URL hash
	tmpFile := filepath.Join(tmpDir, hashURL(downloadURL)+".tmp")

	var offset int64 = 0
	if info, err := os.Stat(tmpFile); err == nil {
		offset = info.Size()
		slog.Info("[download] resuming from partial download",
			"offset_bytes", offset)
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(1<<(attempt-1)) * time.Second
			slog.Warn("[download] retrying after backoff",
				"attempt", attempt+1,
				"total_attempts", maxRetries+1,
				"backoff", backoff.String())
			time.Sleep(backoff)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		req, err := http.NewRequestWithContext(ctx, "GET", downloadURL, nil)
		if err != nil {
			cancel()
			lastErr = &InstallError{Type: ErrNetwork, Message: fmt.Sprintf("create request: %v", err), Err: err}
			continue
		}

		// Add Range header if resuming
		if offset > 0 {
			req.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
		}

		resp, err := downloadClient.Do(req)
		if err != nil {
			cancel()

			// DOWN-03/DOWN-04: Check for timeout and connection errors
			if context.DeadlineExceeded == ctx.Err() {
				slog.Error("[templates] download timeout",
					"url", SanitizeURL(downloadURL),
					"timeout_seconds", 120,
					"error", err.Error())

				lastErr = &InstallError{
					Type: ErrNetwork,
					Message: fmt.Sprintf("下载超时（120 秒）:\n\n"+
						"URL: %s\n\n"+
						"可能的原因:\n"+
						"1. 网络连接不稳定\n"+
						"2. 服务器响应缓慢\n"+
						"3. 防火墙阻止连接\n\n"+
						"建议: 检查网络连接或稍后重试", SanitizeURL(downloadURL)),
					Err: err,
				}
			} else {
				// DOWN-03: Windows-specific firewall/antivirus hints
				isConnectionError := strings.Contains(err.Error(), "connection refused") ||
					strings.Contains(err.Error(), "connection reset") ||
					strings.Contains(err.Error(), "timeout")

				if isConnectionError && runtime.GOOS == "windows" {
					slog.Warn("[templates] Windows network error detected",
						"error", err.Error(),
						"suggestion", "firewall or antivirus may be blocking")

					lastErr = &InstallError{
						Type: ErrNetwork,
						Message: fmt.Sprintf("网络连接失败:\n\n"+
							"错误: %v\n\n"+
							"可能的原因:\n"+
							"1. Windows 防火墙阻止 Presto 访问网络\n"+
							"2. 杀毒软件阻止下载\n"+
							"3. 网络代理配置问题\n\n"+
							"建议:\n"+
							"- 检查 Windows 防火墙设置（允许 Presto）\n"+
							"- 暂时禁用杀毒软件\n"+
							"- 检查代理设置", err),
						Err: err,
					}
				} else {
					lastErr = &InstallError{
						Type:    ErrNetwork,
						Message: fmt.Sprintf("download failed: %v", err),
						Err:     err,
					}
				}
			}

			slog.Warn("[download] attempt failed",
				"attempt", attempt+1,
				"error", err.Error())
			continue
		}

		// Check if server supports Range requests
		if offset > 0 && resp.StatusCode != http.StatusPartialContent {
			slog.Warn("[download] server doesn't support resume, starting fresh",
				"status_code", resp.StatusCode)
			resp.Body.Close()
			cancel()
			offset = 0
			os.Remove(tmpFile)
			continue
		}

		if err := checkHTTPStatus(resp, "download"); err != nil {
			resp.Body.Close()
			cancel()
			if resp.StatusCode >= 400 && resp.StatusCode < 500 {
				return nil, err // Client error, don't retry
			}
			lastErr = err
			continue
		}

		// Open file for appending (or create)
		file, err := os.OpenFile(tmpFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
		if err != nil {
			resp.Body.Close()
			cancel()
			lastErr = &InstallError{Type: ErrServer, Message: fmt.Sprintf("open tmp file: %v", err), Err: err}
			continue
		}

		// Calculate total size
		total := resp.ContentLength
		if offset > 0 {
			total += offset // Total = remaining + already downloaded
		}

		// Wrap with progress reader
		pr := NewProgressReader(resp.Body, total, func(downloaded, total int64) {
			if onProgress != nil {
				onProgress(downloaded+offset, total)
			}
		})

		_, err = io.Copy(file, pr)
		file.Close()
		resp.Body.Close()
		cancel()

		if err != nil {
			slog.Warn("[download] attempt failed",
				"attempt", attempt+1,
				"error", err.Error())
			lastErr = &InstallError{Type: ErrNetwork, Message: fmt.Sprintf("download failed: %v", err), Err: err}
			continue // Keep tmp file for retry
		}

		// Success: read full file
		data, err := os.ReadFile(tmpFile)
		if err != nil {
			lastErr = &InstallError{Type: ErrServer, Message: fmt.Sprintf("read tmp file: %v", err), Err: err}
			continue
		}

		// Clean up on success
		os.Remove(tmpFile)
		slog.Info("[download] completed with resume support",
			"bytes", len(data))
		return data, nil
	}

	return nil, lastErr
}

// CleanupTmpDownloadFiles removes all temporary download files on startup
func CleanupTmpDownloadFiles() {
	tmpDir := filepath.Join(os.TempDir(), tmpDownloadDir)
	if err := os.RemoveAll(tmpDir); err != nil {
		slog.Warn("[download] failed to cleanup tmp dir",
			"error", err.Error())
	} else {
		slog.Info("[download] cleaned up tmp download files")
	}
}

func hashURL(url string) string {
	h := sha256.Sum256([]byte(url))
	return hex.EncodeToString(h[:8])
}
