package template

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

// SEC-20: Custom HTTP client with timeout, replacing http.DefaultClient
var httpClient = &http.Client{
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

// SEC-07: Domain whitelist for download URLs
var allowedDownloadHosts = map[string]bool{
	"github.com":                              true,
	"api.github.com":                          true,
	"objects.githubusercontent.com":           true,
	"github-releases.githubusercontent.com":   true,
	"github.githubassets.com":                 true,
	"codeload.github.com":                     true,
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
	// ExpectedSHA256 is the hex-encoded SHA256 hash from registry.
	// If set, verification is mandatory.
	ExpectedSHA256 string
}

func (m *Manager) Install(owner, repo string, opts *InstallOpts) error {
	// SEC-06/SEC-17: Validate owner and repo names
	if err := validateName(owner); err != nil {
		return fmt.Errorf("invalid owner: %w", err)
	}
	if err := validateName(repo); err != nil {
		return fmt.Errorf("invalid repo: %w", err)
	}

	var downloadURL string
	var expectedHash string

	if opts != nil && opts.DownloadURL != "" {
		// SEC-01: Registry-based install — URL and SHA256 from trusted registry
		downloadURL = opts.DownloadURL
		expectedHash = strings.ToLower(opts.ExpectedSHA256)
		log.Printf("[templates] registry install: %s/%s (SHA256 from registry)", owner, repo)
	} else {
		log.Printf("[templates] discovery install: %s/%s (fetching from release)", owner, repo)

		apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)
		resp, err := httpClient.Get(apiURL)
		if err != nil {
			return fmt.Errorf("fetch release: %w", err)
		}
		defer resp.Body.Close()

		if err := checkHTTPStatus(resp, "fetch release"); err != nil {
			return err
		}

		var release GitHubRelease
		if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
			return fmt.Errorf("decode release: %w", err)
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
			return fmt.Errorf("no binary found for %s/%s", runtime.GOOS, runtime.GOARCH)
		}

		// SEC-01: Try to fetch checksums from release assets (same-source, weaker)
		expectedHash = lookupChecksumFromRelease(release.Assets, assetName)
	}

	// SEC-07: Validate download URL domain
	parsedURL, err := url.Parse(downloadURL)
	if err != nil {
		return fmt.Errorf("invalid download URL: %w", err)
	}
	if !isAllowedDownloadHost(parsedURL.Host) {
		return fmt.Errorf("download URL host not allowed: %s", parsedURL.Host)
	}

	binResp, err := httpClient.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("download binary: %w", err)
	}
	defer binResp.Body.Close()

	if err := checkHTTPStatus(binResp, "download binary"); err != nil {
		return err
	}

	// SEC-13: Limit download to 100MB
	data, err := io.ReadAll(io.LimitReader(binResp.Body, 100<<20))
	if err != nil {
		return fmt.Errorf("read binary: %w", err)
	}

	// SEC-01: Verify checksum if available
	if expectedHash != "" {
		actualHash := sha256.Sum256(data)
		actualHex := hex.EncodeToString(actualHash[:])
		if actualHex != expectedHash {
			return fmt.Errorf("SHA256 mismatch: expected %s, got %s", expectedHash, actualHex)
		}
	}

	tmpFile, err := os.CreateTemp("", "presto-template-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		return fmt.Errorf("write temp binary: %w", err)
	}
	tmpFile.Close()

	// SEC-28: Set executable permission
	if err := os.Chmod(tmpPath, 0700); err != nil {
		return fmt.Errorf("chmod temp binary: %w", err)
	}

	executor := NewExecutor(tmpPath)
	manifestBytes, err := executor.GetManifest()
	if err != nil {
		return fmt.Errorf("get manifest from binary: %w", err)
	}
	manifest, err := ParseManifest(manifestBytes)
	if err != nil {
		return fmt.Errorf("parse manifest: %w", err)
	}

	// SEC-06: Validate manifest name
	if err := validateName(manifest.Name); err != nil {
		return fmt.Errorf("invalid template name from manifest: %w", err)
	}

	// SEC-06: Verify resolved path is within TemplatesDir
	tplDir := filepath.Join(m.TemplatesDir, manifest.Name)
	absTemplatesDir, _ := filepath.Abs(m.TemplatesDir)
	absTplDir, _ := filepath.Abs(tplDir)
	if !strings.HasPrefix(absTplDir, absTemplatesDir+string(filepath.Separator)) {
		return fmt.Errorf("template directory escapes base: %s", absTplDir)
	}

	// SEC-28: Use restrictive permissions
	if err := os.MkdirAll(tplDir, 0700); err != nil {
		return fmt.Errorf("create template dir: %w", err)
	}

	binaryName := templateBinaryName(manifest.Name)

	binPath := filepath.Join(tplDir, binaryName)
	if err := os.WriteFile(binPath, data, 0700); err != nil {
		return fmt.Errorf("write binary: %w", err)
	}

	// Write manifest.json
	// SEC-45: Restrictive file permissions
	if err := os.WriteFile(filepath.Join(tplDir, "manifest.json"), manifestBytes, 0600); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}

	return nil
}

// SEC-01: lookupChecksumFromRelease searches release assets for a checksum file
// and returns the SHA256 hash for the given asset name (same-source, weaker).
func lookupChecksumFromRelease(assets []GitHubAsset, assetName string) string {
	for _, asset := range assets {
		if asset.Name != "checksums.txt" && asset.Name != "SHA256SUMS" {
			continue
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
