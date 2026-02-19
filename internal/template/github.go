package template

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
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
func checkHTTPStatus(resp *http.Response, context string) error {
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("%s: HTTP %d: %s", context, resp.StatusCode, string(body))
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
	// SEC-20: Use custom client with timeout
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

func (m *Manager) Install(owner, repo string) error {
	// SEC-06/SEC-17: Validate owner and repo names
	if err := validateName(owner); err != nil {
		return fmt.Errorf("invalid owner: %w", err)
	}
	if err := validateName(repo); err != nil {
		return fmt.Errorf("invalid repo: %w", err)
	}

	// Fetch latest release metadata (SEC-18, SEC-20)
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

	// Find platform-specific binary asset
	pattern := fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH)
	var downloadURL string
	var assetName string
	for _, asset := range release.Assets {
		if strings.Contains(asset.Name, pattern) {
			downloadURL = asset.BrowserDownloadURL
			assetName = asset.Name
			break
		}
	}
	if downloadURL == "" {
		return fmt.Errorf("no binary found for %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	// SEC-07: Validate download URL domain
	parsedURL, err := url.Parse(downloadURL)
	if err != nil {
		return fmt.Errorf("invalid download URL: %w", err)
	}
	if !isAllowedDownloadHost(parsedURL.Host) {
		return fmt.Errorf("download URL host not allowed: %s", parsedURL.Host)
	}

	// SEC-01: Try to fetch checksums from release assets
	var expectedHash string
	for _, asset := range release.Assets {
		if asset.Name == "checksums.txt" || asset.Name == "SHA256SUMS" {
			checksumResp, cerr := httpClient.Get(asset.BrowserDownloadURL)
			if cerr == nil {
				defer checksumResp.Body.Close()
				if checksumResp.StatusCode >= 200 && checksumResp.StatusCode < 300 {
					body, _ := io.ReadAll(io.LimitReader(checksumResp.Body, 1<<20))
					for _, line := range strings.Split(string(body), "\n") {
						fields := strings.Fields(line)
						if len(fields) >= 2 {
							fname := strings.TrimPrefix(fields[1], "*")
							if fname == assetName {
								expectedHash = strings.ToLower(fields[0])
								break
							}
						}
					}
				}
			}
			break
		}
	}

	// SEC-06: Derive and validate template name
	name := repo
	if strings.HasPrefix(name, "presto-template-") {
		name = name[len("presto-template-"):]
	}
	name = filepath.Base(name)
	if err := validateName(name); err != nil {
		return fmt.Errorf("invalid template name: %w", err)
	}

	// SEC-06: Verify resolved path is within TemplatesDir
	tplDir := filepath.Join(m.TemplatesDir, name)
	absTemplatesDir, _ := filepath.Abs(m.TemplatesDir)
	absTplDir, _ := filepath.Abs(tplDir)
	if !strings.HasPrefix(absTplDir, absTemplatesDir+string(filepath.Separator)) {
		return fmt.Errorf("template directory escapes base: %s", absTplDir)
	}

	// SEC-28: Use restrictive permissions
	if err := os.MkdirAll(tplDir, 0700); err != nil {
		return fmt.Errorf("create template dir: %w", err)
	}

	// Download binary (SEC-18, SEC-20)
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
		actualHashHex := hex.EncodeToString(actualHash[:])
		if actualHashHex != expectedHash {
			return fmt.Errorf("SHA256 mismatch: expected %s, got %s", expectedHash, actualHashHex)
		}
	}

	// SEC-28: Use restrictive permissions for binary
	binPath := filepath.Join(tplDir, repo)
	if err := os.WriteFile(binPath, data, 0700); err != nil {
		return fmt.Errorf("write binary: %w", err)
	}

	return nil
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

	// Verify target exists and is a directory
	info, err := os.Stat(tplDir)
	if err != nil {
		return fmt.Errorf("template not found: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("not a directory: %s", name)
	}

	return os.RemoveAll(tplDir)
}
