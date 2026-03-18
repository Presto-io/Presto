package template

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

// InstallErrorType represents the type of installation error
type InstallErrorType string

const (
	ErrNetwork         InstallErrorType = "network_error"
	ErrNotFound        InstallErrorType = "not_found"
	ErrChecksumMismatch InstallErrorType = "checksum_mismatch"
	ErrServer          InstallErrorType = "server_error"
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
	"github.com":                              true,
	"api.github.com":                          true,
	"objects.githubusercontent.com":           true,
	"github-releases.githubusercontent.com":   true,
	"github.githubassets.com":                 true,
	"codeload.github.com":                     true,
	"release-assets.githubusercontent.com":    true,
	"presto.c-1o.top":                         true,
	"cdn.presto.c-1o.top":                     true,
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
	// If set, skips GitHub release API lookup.
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

func (m *Manager) Install(owner, repo string, opts *InstallOpts) error {
	startTime := time.Now()

	// SEC-06/SEC-17: Validate owner and repo names
	if err := validateName(owner); err != nil {
		return &InstallError{
			Type:    ErrNotFound,
			Message: fmt.Sprintf("invalid owner: %v", err),
			Err:     err,
		}
	}
	if err := validateName(repo); err != nil {
		return &InstallError{
			Type:    ErrNotFound,
			Message: fmt.Sprintf("invalid repo: %v", err),
			Err:     err,
		}
	}

	var downloadURL string
	var expectedHash string

	if opts != nil && opts.DownloadURL != "" {
		// SEC-01: Registry-based install — URL and SHA256 from trusted registry
		expectedHash = strings.ToLower(opts.ExpectedSHA256)

		// GitHub-first logic with 3s probe
		gitHubReachable := isGitHubReachable()

		if gitHubReachable {
			// Try GitHub first
			log.Printf("[templates] downloading %s/%s from GitHub: %s", owner, repo, opts.DownloadURL)
			downloadURL = opts.DownloadURL

			data, err := downloadWithResume(downloadURL, 3, opts.OnProgress)
			if err != nil {
				// GitHub failed, try CDN fallback
				if opts.CdnURL != "" {
					log.Printf("[templates] GitHub failed: %v, falling back to CDN: %s", err, opts.CdnURL)
					downloadURL = opts.CdnURL
					data, err = downloadWithResume(downloadURL, 3, opts.OnProgress)
					if err != nil {
						return err
					}
				} else {
					return err
				}
			}

			// Continue with installation using downloaded data
			return m.completeInstall(owner, repo, data, expectedHash, opts, startTime)
		} else {
			// GitHub probe failed, go directly to CDN
			if opts.CdnURL != "" {
				log.Printf("[templates] GitHub unreachable, downloading %s/%s from CDN: %s", owner, repo, opts.CdnURL)
				downloadURL = opts.CdnURL
			} else {
				log.Printf("[templates] GitHub unreachable, downloading %s/%s from GitHub URL: %s", owner, repo, opts.DownloadURL)
				downloadURL = opts.DownloadURL
			}

			data, err := downloadWithResume(downloadURL, 3, opts.OnProgress)
			if err != nil {
				return err
			}

			// Continue with installation using downloaded data
			return m.completeInstall(owner, repo, data, expectedHash, opts, startTime)
		}
	} else {
		// Discovery install path (no opts)
		log.Printf("[templates] discovery install: %s/%s (fetching from release)", owner, repo)

		apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)
		resp, err := httpClient.Get(apiURL)
		if err != nil {
			return &InstallError{
				Type:    ErrNetwork,
				Message: fmt.Sprintf("fetch release: %v", err),
				Err:     err,
			}
		}
		defer resp.Body.Close()

		if err := checkHTTPStatus(resp, "fetch release"); err != nil {
			return &InstallError{
				Type:    ErrNotFound,
				Message: fmt.Sprintf("fetch release: %v", err),
				Err:     err,
			}
		}

		var release GitHubRelease
		if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
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
			return &InstallError{
				Type:    ErrNotFound,
				Message: fmt.Sprintf("no binary found for %s/%s", runtime.GOOS, runtime.GOARCH),
				Err:     fmt.Errorf("no binary for platform"),
			}
		}

		// SEC-01: Try to fetch checksums from release assets (same-source, weaker)
		expectedHash = lookupChecksumFromRelease(release.Assets, assetName)
		if expectedHash == "" {
			log.Printf("[security] WARNING: no checksums.txt/SHA256SUMS found in release for %s/%s", owner, repo)
		}

		// Download with retry
		log.Printf("[templates] downloading %s/%s from %s", owner, repo, downloadURL)
		data, err := downloadWithResume(downloadURL, 3, nil)
		if err != nil {
			return err
		}

		// Continue with installation using downloaded data
		return m.completeInstall(owner, repo, data, expectedHash, nil, startTime)
	}
}

// completeInstall finishes the installation process after download
func (m *Manager) completeInstall(owner, repo string, data []byte, expectedHash string, opts *InstallOpts, startTime time.Time) error {
	// SEC-30: 下载→验证→执行 三步流程
	// Step 1: 下载已完成（data 已获取）
	// Step 2: 验证 SHA256
	actualHash := sha256.Sum256(data)
	actualHex := hex.EncodeToString(actualHash[:])

	if expectedHash == "" {
		// SEC-01: official 和 verified 模板必须有 SHA256，缺失则拒绝安装
		if opts != nil && (opts.Trust == "official" || opts.Trust == "verified") {
			return &InstallError{
				Type:    ErrChecksumMismatch,
				Message: fmt.Sprintf("SHA256 required for %s templates but not found", opts.Trust),
				Err:     fmt.Errorf("missing SHA256"),
			}
		}
		// SEC-30: 社区模板无校验时记录警告
		log.Printf("[security] WARNING: installing %s/%s without SHA256 verification (hash: %s)", owner, repo, actualHex)
	} else {
		if actualHex != expectedHash {
			log.Printf("[templates] SHA256 mismatch for %s/%s: expected %s, got %s", owner, repo, expectedHash, actualHex)
			return &InstallError{
				Type:    ErrChecksumMismatch,
				Message: fmt.Sprintf("SHA256 mismatch: expected %s, got %s", expectedHash, actualHex),
				Err:     fmt.Errorf("checksum mismatch"),
			}
		}
		log.Printf("[templates] SHA256 verified for %s/%s: %s", owner, repo, actualHex)
	}

	// Step 3: 验证通过后才执行二进制
	tmpFile, err := os.CreateTemp("", "presto-template-*")
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

	// SEC-28: Use restrictive permissions
	if err := os.MkdirAll(tplDir, 0700); err != nil {
		return &InstallError{
			Type:    ErrServer,
			Message: fmt.Sprintf("create template dir: %v", err),
			Err:     err,
		}
	}

	binaryName := templateBinaryName(manifest.Name)

	binPath := filepath.Join(tplDir, binaryName)
	if err := os.WriteFile(binPath, data, 0700); err != nil {
		return &InstallError{
			Type:    ErrServer,
			Message: fmt.Sprintf("write binary: %v", err),
			Err:     err,
		}
	}

	// Write manifest.json
	// SEC-45: Restrictive file permissions
	if err := os.WriteFile(filepath.Join(tplDir, "manifest.json"), manifestBytes, 0600); err != nil {
		return &InstallError{
			Type:    ErrServer,
			Message: fmt.Sprintf("write manifest: %v", err),
			Err:     err,
		}
	}

	// Log completion with duration
	duration := time.Since(startTime)
	log.Printf("[templates] installed %s/%s successfully in %s", owner, repo, duration)

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
			log.Printf("[security] BLOCKED: checksum URL host not in whitelist: %s", asset.BrowserDownloadURL)
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
		log.Printf("[templates] GitHub probe failed: %v", err)
		return false
	}

	resp, err := probeClient.Do(req)
	if err != nil {
		log.Printf("[templates] GitHub probe failed: %v", err)
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		log.Printf("[templates] GitHub probe succeeded")
		return true
	}

	log.Printf("[templates] GitHub probe failed: HTTP %d", resp.StatusCode)
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
		log.Printf("[security] BLOCKED: download URL host not in whitelist: %s (full URL: %s)", parsedURL.Host, downloadURL)
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
			log.Printf("[templates] download attempt %d/%d: waiting %v before retry", attempt+1, maxRetries+1, backoff)
			time.Sleep(backoff)
		}

		log.Printf("[templates] download attempt %d/%d: %s", attempt+1, maxRetries+1, downloadURL)

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
			// Network error - retry
			lastErr = &InstallError{
				Type:    ErrNetwork,
				Message: fmt.Sprintf("download failed: %v", err),
				Err:     err,
			}
			log.Printf("[templates] download attempt %d failed: %v", attempt+1, err)
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
			log.Printf("[templates] download attempt %d failed with server error: HTTP %d", attempt+1, statusCode)
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
			log.Printf("[templates] download attempt %d failed: %v", attempt+1, err)
			continue
		}

		// Success
		log.Printf("[templates] download succeeded: %d bytes", len(data))
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
