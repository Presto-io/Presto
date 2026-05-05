package template

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

// InstallErrorType represents the type of installation error
type InstallErrorType string

const (
	ErrNetwork          InstallErrorType = "network_error"
	ErrNotFound         InstallErrorType = "not_found"
	ErrChecksumMismatch InstallErrorType = "checksum_mismatch"
	ErrServer           InstallErrorType = "server_error"
)

// InstallError wraps installation errors with type classification
type InstallError struct {
	Type    InstallErrorType
	Message string
	Err     error
}

func (e *InstallError) Error() string { return e.Message }
func (e *InstallError) Unwrap() error { return e.Err }

// SEC-20: Separate HTTP clients for different purposes
var (
	// probeClient: 3s timeout for GitHub connectivity check
	probeClient = &http.Client{
		Timeout: 3 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if !isAllowedDownloadHost(req.URL.Host) {
				return fmt.Errorf("redirect to disallowed host: %s", req.URL.Host)
			}
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	// downloadClient: Custom Transport with split timeouts
	// Connection: 10s, Response header: 15s, Overall: 120s
	downloadClient = &http.Client{
		Timeout: 120 * time.Second, // Overall timeout for large files
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				// SEC-20: Connection timeout 10s
				d := net.Dialer{Timeout: 10 * time.Second}
				return d.DialContext(ctx, network, addr)
			},
			ResponseHeaderTimeout: 15 * time.Second, // Response header timeout
			DisableKeepAlives:     false,
			MaxIdleConns:          10,
			IdleConnTimeout:       30 * time.Second,
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// SEC-07: Only allow redirects to known safe domains
			if !isAllowedDownloadHost(req.URL.Host) {
				return fmt.Errorf("redirect to disallowed host: %s", req.URL.Host)
			}
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	// Legacy httpClient kept for non-download operations (discovery, registry)
	httpClient = &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// SEC-07: Only allow redirects to known GitHub domains
			if !isAllowedDownloadHost(req.URL.Host) {
				return fmt.Errorf("redirect to disallowed host: %s", req.URL.Host)
			}
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}
)

// SEC-07: Domain whitelist for download URLs
var allowedDownloadHosts = map[string]bool{
	"github.com":                            true,
	"api.github.com":                        true,
	"objects.githubusercontent.com":         true,
	"github-releases.githubusercontent.com": true,
	"github.githubassets.com":               true,
	"codeload.github.com":                   true,
	"release-assets.githubusercontent.com":  true,
	"presto.c-1o.top":                       true,
	"cdn.presto.c-1o.top":                   true,
}

func isAllowedDownloadHost(host string) bool {
	return allowedDownloadHosts[host]
}

// SEC-06: Name validation regex for owner/repo/template names
var validNameRe = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)

func validateName(name string) error {
	if name == "" {
		return fmt.Errorf("name is empty")
	}
	if !validNameRe.MatchString(name) {
		return fmt.Errorf("name contains invalid characters: %q", name)
	}
	if strings.Contains(name, "..") {
		return fmt.Errorf("name contains path traversal: %q", name)
	}
	return nil
}

// SEC-18: HTTP response status validation
func checkHTTPStatus(resp *http.Response, op string) error {
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("%s: HTTP %d: %s", op, resp.StatusCode, string(body))
	}
	return nil
}

type GitHubRepo struct {
	FullName    string `json:"full_name"`
	Description string `json:"description"`
	HTMLURL     string `json:"html_url"`
	Owner       struct {
		Login string `json:"login"`
	} `json:"owner"`
	Name string `json:"name"`
}

type GitHubSearchResult struct {
	Items []GitHubRepo `json:"items"`
}

type GitHubRelease struct {
	TagName string        `json:"tag_name"`
	Assets  []GitHubAsset `json:"assets"`
}

type GitHubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

func DiscoverTemplates() ([]GitHubRepo, error) {
	resp, err := httpClient.Get("https://api.github.com/search/repositories?q=topic:presto-template&sort=stars")
	if err != nil {
		return nil, fmt.Errorf("discover templates: %w", err)
	}
	defer resp.Body.Close()
	// SEC-18: Check HTTP status code
	if err := checkHTTPStatus(resp, "discover templates"); err != nil {
		return nil, err
	}

	var result GitHubSearchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode discover response: %w", err)
	}
	return result.Items, nil
}

// InstallOpts provides optional registry-sourced download info.
// SEC-01: When provided, SHA256 comes from registry (separate trusted source)
// instead of from the same GitHub release (untrusted same-source).
type InstallOpts struct {
	// DownloadURL is the direct binary download URL from registry.
	// If set with CdnURL, it is used as a fallback after the CDN.
	DownloadURL string
	// CdnURL is the CDN mirror URL for the binary (bypasses GitHub).
	// If set, tried first before DownloadURL.
	CdnURL string
	// ExpectedSHA256 is the hex-encoded SHA256 hash from registry.
	// If set, verification is mandatory.
	ExpectedSHA256 string
	// Trust is the trust level: "official", "verified", "community", or "".
	Trust string
	// OnProgress is an optional callback for download progress updates.
	// If set, called periodically with bytes downloaded and total bytes.
	OnProgress ProgressCallback
}

type downloadCandidate struct {
	source   string
	url      string
	filename string
}

func downloadCandidates(opts *InstallOpts) []downloadCandidate {
	if opts == nil {
		return nil
	}

	var candidates []downloadCandidate
	if opts.CdnURL != "" {
		candidates = append(candidates, downloadCandidate{source: "cdn", url: opts.CdnURL, filename: downloadFilename(opts.CdnURL)})
	}
	if opts.DownloadURL != "" {
		candidates = append(candidates, downloadCandidate{source: "github", url: opts.DownloadURL, filename: downloadFilename(opts.DownloadURL)})
	}
	return candidates
}

func downloadFilename(downloadURL string) string {
	parsed, err := url.Parse(downloadURL)
	if err != nil {
		return ""
	}
	name := path.Base(parsed.Path)
	if name == "." || name == "/" {
		return ""
	}
	return name
}

func (m *Manager) Install(owner, repo string, opts *InstallOpts) error {
	startTime := time.Now()

	// SEC-06/SEC-17: Validate owner and repo names
	if err := validateName(owner); err != nil {
		slog.Error("[templates] invalid owner name",
			"error_type", string(ErrNotFound),
			"owner", owner,
			"error", err.Error())
		return &InstallError{
			Type:    ErrNotFound,
			Message: fmt.Sprintf("invalid owner: %v", err),
			Err:     err,
		}
	}
	if err := validateName(repo); err != nil {
		slog.Error("[templates] invalid repo name",
			"error_type", string(ErrNotFound),
			"repo", repo,
			"error", err.Error())
		return &InstallError{
			Type:    ErrNotFound,
			Message: fmt.Sprintf("invalid repo: %v", err),
			Err:     err,
		}
	}

	var downloadURL string
	var expectedHash string

	if opts != nil && (opts.DownloadURL != "" || opts.CdnURL != "") {
		// SEC-01: Registry-based install — URL and SHA256 from trusted registry
		expectedHash = strings.ToLower(opts.ExpectedSHA256)

		candidates := downloadCandidates(opts)
		var lastErr error
		for i, candidate := range candidates {
			slog.Info("[templates] starting download",
				"owner", owner,
				"repo", repo,
				"source", candidate.source,
				"url", SanitizeURL(candidate.url))

			data, err := downloadWithResume(candidate.url, 3, opts.OnProgress)
			if err == nil {
				return m.completeInstall(owner, repo, data, expectedHash, opts, startTime, candidate.filename)
			}

			lastErr = err
			if i+1 < len(candidates) {
				slog.Warn("[templates] download failed, falling back",
					"source", candidate.source,
					"error", err.Error(),
					"next_source", candidates[i+1].source)
			}
		}

		if lastErr != nil {
			return lastErr
		}
		return &InstallError{
			Type:    ErrNotFound,
			Message: "no download URL available",
			Err:     fmt.Errorf("no download URL available"),
		}
	} else {
		// Discovery install path (no opts)
		slog.Info("[templates] discovery install",
			"owner", owner,
			"repo", repo)

		apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)
		resp, err := httpClient.Get(apiURL)
		if err != nil {
			slog.Error("[templates] fetch release failed",
				"error_type", string(ErrNetwork),
				"owner", owner,
				"repo", repo,
				"error", err.Error())
			return &InstallError{
				Type:    ErrNetwork,
				Message: fmt.Sprintf("fetch release: %v", err),
				Err:     err,
			}
		}
		defer resp.Body.Close()

		if err := checkHTTPStatus(resp, "fetch release"); err != nil {
			slog.Error("[templates] fetch release HTTP error",
				"error_type", string(ErrNotFound),
				"owner", owner,
				"repo", repo,
				"error", err.Error())
			return &InstallError{
				Type:    ErrNotFound,
				Message: fmt.Sprintf("fetch release: %v", err),
				Err:     err,
			}
		}

		var release GitHubRelease
		if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
			slog.Error("[templates] decode release failed",
				"error_type", string(ErrServer),
				"owner", owner,
				"repo", repo,
				"error", err.Error())
			return &InstallError{
				Type:    ErrServer,
				Message: fmt.Sprintf("decode release: %v", err),
				Err:     err,
			}
		}

		// Find platform-specific binary asset (try both dash and underscore separators)
		platform := runtime.GOOS + "-" + runtime.GOARCH
		platformAlt := runtime.GOOS + "_" + runtime.GOARCH
		var assetName string
	assetSearch:
		for _, asset := range release.Assets {
			if strings.Contains(asset.Name, platform) || strings.Contains(asset.Name, platformAlt) {
				downloadURL = asset.BrowserDownloadURL
				assetName = asset.Name
				break assetSearch
			}
		}
		if downloadURL == "" {
			slog.Error("[templates] no binary found for platform",
				"error_type", string(ErrNotFound),
				"owner", owner,
				"repo", repo,
				"os", runtime.GOOS,
				"arch", runtime.GOARCH)
			return &InstallError{
				Type:    ErrNotFound,
				Message: fmt.Sprintf("no binary found for %s/%s", runtime.GOOS, runtime.GOARCH),
				Err:     fmt.Errorf("no binary for platform"),
			}
		}

		// SEC-01: Try to fetch checksums from release assets (same-source, weaker)
		expectedHash = lookupChecksumFromRelease(release.Assets, assetName)
		if expectedHash == "" {
			slog.Warn("[security] no checksum found in release",
				"owner", owner,
				"repo", repo)
		}

		// Download with retry
		slog.Info("[templates] downloading binary",
			"owner", owner,
			"repo", repo,
			"url", SanitizeURL(downloadURL))
		data, err := downloadWithResume(downloadURL, 3, nil)
		if err != nil {
			return err
		}

		// Continue with installation using downloaded data
		return m.completeInstall(owner, repo, data, expectedHash, nil, startTime, assetName)
	}
}

// completeInstall finishes the installation process after download.
func (m *Manager) completeInstall(owner, repo string, data []byte, expectedHash string, opts *InstallOpts, startTime time.Time, downloadedFilename string) error {
	// SEC-30: 下载→验证→执行 三步流程
	// Step 1: 下载已完成（data 已获取）
	// Step 2: 验证 SHA256
	actualHash := sha256.Sum256(data)
	actualHex := hex.EncodeToString(actualHash[:])

	if expectedHash == "" {
		// SEC-01: official 和 verified 模板必须有 SHA256，缺失则拒绝安装
		if opts != nil && (opts.Trust == "official" || opts.Trust == "verified") {
			slog.Error("[templates] SHA256 required for trusted template",
				"error_type", string(ErrChecksumMismatch),
				"owner", owner,
				"repo", repo,
				"trust", opts.Trust)
			return &InstallError{
				Type:    ErrChecksumMismatch,
				Message: fmt.Sprintf("SHA256 required for %s templates but not found", opts.Trust),
				Err:     fmt.Errorf("missing SHA256"),
			}
		}
		// SEC-30: 社区模板无校验时记录警告
		slog.Warn("[security] installing without SHA256 verification",
			"owner", owner,
			"repo", repo,
			"hash", actualHex)
	} else {
		if actualHex != expectedHash {
			slog.Error("[templates] SHA256 mismatch",
				"owner", owner,
				"repo", repo,
				"expected", expectedHash,
				"actual", actualHex)
			return &InstallError{
				Type:    ErrChecksumMismatch,
				Message: fmt.Sprintf("SHA256 mismatch: expected %s, got %s", expectedHash, actualHex),
				Err:     fmt.Errorf("checksum mismatch"),
			}
		}
		slog.Info("[templates] SHA256 verified",
			"owner", owner,
			"repo", repo,
			"hash", actualHex)
	}

	// Step 3: 验证通过后才执行二进制
	// FIX: Windows requires .exe suffix for executable files
	tmpPattern := "presto-template-*"
	if runtime.GOOS == "windows" {
		tmpPattern = "presto-template-*.exe"
	}
	tmpFile, err := os.CreateTemp("", tmpPattern)
	if err != nil {
		return &InstallError{
			Type:    ErrServer,
			Message: fmt.Sprintf("create temp file: %v", err),
			Err:     err,
		}
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		return &InstallError{
			Type:    ErrServer,
			Message: fmt.Sprintf("write temp binary: %v", err),
			Err:     err,
		}
	}
	tmpFile.Close()

	// SEC-28: Set executable permission
	if err := os.Chmod(tmpPath, 0700); err != nil {
		return &InstallError{
			Type:    ErrServer,
			Message: fmt.Sprintf("chmod temp binary: %v", err),
			Err:     err,
		}
	}

	executor := NewExecutor(tmpPath)
	manifestBytes, err := executor.GetManifest()
	if err != nil {
		return &InstallError{
			Type:    ErrServer,
			Message: fmt.Sprintf("get manifest from binary: %v", err),
			Err:     err,
		}
	}
	manifest, err := ParseManifest(manifestBytes)
	if err != nil {
		return &InstallError{
			Type:    ErrServer,
			Message: fmt.Sprintf("parse manifest: %v", err),
			Err:     err,
		}
	}

	// SEC-06: Validate manifest name
	if err := validateName(manifest.Name); err != nil {
		return &InstallError{
			Type:    ErrNotFound,
			Message: fmt.Sprintf("invalid template name from manifest: %v", err),
			Err:     err,
		}
	}

	// SEC-06: Verify resolved path is within TemplatesDir
	tplDir := filepath.Join(m.TemplatesDir, manifest.Name)
	absTemplatesDir, _ := filepath.Abs(m.TemplatesDir)
	absTplDir, _ := filepath.Abs(tplDir)
	if !strings.HasPrefix(absTplDir, absTemplatesDir+string(filepath.Separator)) {
		return &InstallError{
			Type:    ErrNotFound,
			Message: fmt.Sprintf("template directory escapes base: %s", absTplDir),
			Err:     fmt.Errorf("path traversal"),
		}
	}

	// DOWN-01: Windows-specific path validation (MAX_PATH, invalid chars, reserved names)
	if runtime.GOOS == "windows" {
		if err := validateWindowsPath(tplDir); err != nil {
			slog.Error("[templates] Windows path validation failed",
				"path", tplDir,
				"error", err.Error())
			return &InstallError{
				Type:    ErrServer,
				Message: fmt.Sprintf("Windows path error: %v", err),
				Err:     err,
			}
		}
	}

	// SEC-28: Use restrictive permissions
	if err := os.MkdirAll(tplDir, 0700); err != nil {
		slog.Error("[templates] failed to create template directory",
			"path", tplDir,
			"error", err.Error())

		// DOWN-02: Windows-specific permission error detection
		if runtime.GOOS == "windows" && isWindowsPermissionError(err) {
			return &InstallError{
				Type:    ErrServer,
				Message: windowsPermissionErrorMsg(tplDir),
				Err:     err,
			}
		}

		return &InstallError{
			Type:    ErrServer,
			Message: fmt.Sprintf("create template dir: %v", err),
			Err:     err,
		}
	}

	binaryName, writeManifest, err := installArtifactLayout(runtime.GOOS, manifest.Name, downloadedFilename)
	if err != nil {
		return &InstallError{
			Type:    ErrServer,
			Message: fmt.Sprintf("resolve install artifact layout: %v", err),
			Err:     err,
		}
	}

	binPath := filepath.Join(tplDir, binaryName)
	if err := os.WriteFile(binPath, data, 0700); err != nil {
		slog.Error("[templates] failed to write binary",
			"path", binPath,
			"error", err.Error())

		// DOWN-02: Windows-specific permission error detection
		if runtime.GOOS == "windows" && isWindowsPermissionError(err) {
			return &InstallError{
				Type:    ErrServer,
				Message: windowsPermissionErrorMsg(binPath),
				Err:     err,
			}
		}

		return &InstallError{
			Type:    ErrServer,
			Message: fmt.Sprintf("write binary: %v", err),
			Err:     err,
		}
	}

	if writeManifest {
		// Write manifest.json
		// SEC-45: Restrictive file permissions
		manifestPath := filepath.Join(tplDir, "manifest.json")
		if err := os.WriteFile(manifestPath, manifestBytes, 0600); err != nil {
			slog.Error("[templates] failed to write manifest",
				"path", manifestPath,
				"error", err.Error())

			// DOWN-02: Windows-specific permission error detection
			if runtime.GOOS == "windows" && isWindowsPermissionError(err) {
				return &InstallError{
					Type:    ErrServer,
					Message: windowsPermissionErrorMsg(manifestPath),
					Err:     err,
				}
			}

			return &InstallError{
				Type:    ErrServer,
				Message: fmt.Sprintf("write manifest: %v", err),
				Err:     err,
			}
		}
	} else {
		legacyManifestPath := filepath.Join(tplDir, "manifest.json")
		if err := os.Remove(legacyManifestPath); err != nil && !os.IsNotExist(err) {
			return &InstallError{
				Type:    ErrServer,
				Message: fmt.Sprintf("remove legacy manifest: %v", err),
				Err:     err,
			}
		}

		legacyBinaryPath := filepath.Join(tplDir, templateBinaryName(manifest.Name))
		if legacyBinaryPath != binPath {
			if err := os.Remove(legacyBinaryPath); err != nil && !os.IsNotExist(err) {
				return &InstallError{
					Type:    ErrServer,
					Message: fmt.Sprintf("remove legacy binary: %v", err),
					Err:     err,
				}
			}
		}
	}

	// Log completion with duration
	duration := time.Since(startTime)
	slog.Info("[templates] installed successfully",
		"owner", owner,
		"repo", repo,
		"duration", duration.String(),
		"duration_ms", duration.Milliseconds())

	return nil
}

// SEC-01: lookupChecksumFromRelease searches release assets for a checksum file
// and returns the SHA256 hash for the given asset name (same-source, weaker).
func lookupChecksumFromRelease(assets []GitHubAsset, assetName string) string {
	for _, asset := range assets {
		if asset.Name != "checksums.txt" && asset.Name != "SHA256SUMS" {
			continue
		}
		// SEC-07: Validate checksum file URL domain
		if checksumURL, err := url.Parse(asset.BrowserDownloadURL); err != nil || !isAllowedDownloadHost(checksumURL.Host) {
			slog.Warn("[security] blocked checksum URL host not in whitelist",
				"url", SanitizeURL(asset.BrowserDownloadURL))
			return ""
		}
		resp, err := httpClient.Get(asset.BrowserDownloadURL)
		if err != nil {
			return ""
		}
		// SEC-47: Close body immediately instead of defer in loop
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			resp.Body.Close()
			return ""
		}
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		resp.Body.Close()
		for _, line := range strings.Split(string(body), "\n") {
			fields := strings.Fields(line)
			if len(fields) >= 2 && strings.TrimPrefix(fields[1], "*") == assetName {
				return strings.ToLower(fields[0])
			}
		}
		return ""
	}
	return ""
}

// isGitHubReachable probes GitHub connectivity with 3s timeout
// Returns true if GitHub responds, false otherwise
func isGitHubReachable() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "HEAD", "https://api.github.com/zen", nil)
	if err != nil {
		slog.Debug("[templates] GitHub probe failed",
			"error", err.Error())
		return false
	}

	resp, err := probeClient.Do(req)
	if err != nil {
		slog.Debug("[templates] GitHub probe failed",
			"error", err.Error())
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		slog.Debug("[templates] GitHub probe succeeded")
		return true
	}

	slog.Debug("[templates] GitHub probe failed",
		"status_code", resp.StatusCode)
	return false
}

// downloadWithRetry downloads a URL with retry logic and exponential backoff
// maxRetries: maximum number of retry attempts (total attempts = maxRetries + 1)
// onProgress: optional callback for download progress updates (can be nil)
// Returns downloaded data or error
func downloadWithRetry(downloadURL string, maxRetries int, onProgress ProgressCallback) ([]byte, error) {
	// SEC-07: Validate URL domain
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

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 1s, 2s, 4s
			backoff := time.Duration(1<<(attempt-1)) * time.Second
			slog.Warn("[templates] download failed, retrying",
				"attempt", attempt+1,
				"total_attempts", maxRetries+1,
				"backoff", backoff.String())
			time.Sleep(backoff)
		}

		slog.Info("[templates] download attempt",
			"attempt", attempt+1,
			"total_attempts", maxRetries+1,
			"url", SanitizeURL(downloadURL))

		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		req, err := http.NewRequestWithContext(ctx, "GET", downloadURL, nil)
		if err != nil {
			cancel()
			lastErr = &InstallError{
				Type:    ErrNetwork,
				Message: fmt.Sprintf("create request failed: %v", err),
				Err:     err,
			}
			continue
		}

		resp, err := downloadClient.Do(req)
		// Note: cancel() deferred to after response body is read

		if err != nil {
			cancel() // Cancel on error

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
					// Network error - retry
					lastErr = &InstallError{
						Type:    ErrNetwork,
						Message: fmt.Sprintf("download failed: %v", err),
						Err:     err,
					}
				}
			}

			slog.Warn("[templates] download attempt failed",
				"attempt", attempt+1,
				"error", err.Error())
			continue
		}

		// Check HTTP status
		if err := checkHTTPStatus(resp, "download binary"); err != nil {
			resp.Body.Close()
			cancel() // Cancel after closing body

			// Classify error
			statusCode := resp.StatusCode
			if statusCode >= 400 && statusCode < 500 {
				// Client errors (4xx) - not transient, don't retry
				return nil, &InstallError{
					Type:    ErrNotFound,
					Message: fmt.Sprintf("download failed: %v", err),
					Err:     err,
				}
			}

			// Server errors (5xx) - transient, retry
			lastErr = &InstallError{
				Type:    ErrServer,
				Message: fmt.Sprintf("server error: %v", err),
				Err:     err,
			}
			slog.Warn("[templates] download attempt failed with server error",
				"attempt", attempt+1,
				"status_code", statusCode)
			continue
		}

		// Read response body with progress tracking
		var data []byte
		if onProgress != nil && resp.ContentLength > 0 {
			// Use ProgressReader to track download progress
			pr := NewProgressReader(resp.Body, resp.ContentLength, onProgress)
			data, err = io.ReadAll(pr)
		} else {
			// No progress callback or unknown content length, read directly
			data, err = io.ReadAll(resp.Body)
		}
		resp.Body.Close()
		cancel() // Cancel after reading and closing body

		if err != nil {
			lastErr = &InstallError{
				Type:    ErrNetwork,
				Message: fmt.Sprintf("read response failed: %v", err),
				Err:     err,
			}
			slog.Warn("[templates] download attempt failed",
				"attempt", attempt+1,
				"error", err.Error())
			continue
		}

		// Success
		slog.Info("[templates] download completed",
			"bytes", len(data))
		return data, nil
	}

	// All retries exhausted
	return nil, lastErr
}

// SEC-05: Secure uninstall with path traversal protection
func (m *Manager) Uninstall(name string) error {
	name = filepath.Base(name)
	if err := validateName(name); err != nil {
		return fmt.Errorf("invalid template name: %w", err)
	}

	tplDir := filepath.Join(m.TemplatesDir, name)

	// Verify resolved path is within TemplatesDir
	absTemplatesDir, _ := filepath.Abs(m.TemplatesDir)
	absTplDir, _ := filepath.Abs(tplDir)
	if !strings.HasPrefix(absTplDir, absTemplatesDir+string(filepath.Separator)) {
		return fmt.Errorf("path escapes templates directory")
	}

	// SEC-38: Use Lstat to detect symlinks (not follow them)
	info, err := os.Lstat(tplDir)
	if err != nil {
		return fmt.Errorf("template not found: %w", err)
	}
	// SEC-38: Reject symlinks to prevent TOCTOU attacks
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refusing to remove symlink: %s", name)
	}
	if !info.IsDir() {
		return fmt.Errorf("not a directory: %s", name)
	}

	return os.RemoveAll(tplDir)
}
