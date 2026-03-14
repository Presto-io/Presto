package template

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

const (
	defaultRegistryURL = "https://presto.c-1o.top/templates/registry.json"
	registryCacheFile  = "registry-cache.json"
	cacheTTL           = 1 * time.Hour
	fetchTimeout       = 15 * time.Second
	// SEC-13: limit registry response size
	maxRegistrySize = 10 << 20 // 10 MB
)

type RegistryPlatformInfo struct {
	URL    string `json:"url"`
	CdnURL string `json:"cdn_url,omitempty"`
	SHA256 string `json:"sha256"`
}

type RegistryEntry struct {
	Name      string                          `json:"name"`
	Version   string                          `json:"version"`
	Trust     string                          `json:"trust"`
	Repo      string                          `json:"repo"`
	Platforms map[string]RegistryPlatformInfo `json:"platforms"`
}

type Registry struct {
	Version   int             `json:"version"`
	UpdatedAt string          `json:"updatedAt"`
	Templates []RegistryEntry `json:"templates"`
}

// VerifyResult describes the outcome of SHA256 verification against the registry.
type VerifyResult string

const (
	VerifyMatched       VerifyResult = "verified"
	VerifyNotInRegistry VerifyResult = "not_in_registry"
	VerifyPending       VerifyResult = "pending"
	VerifyMismatch      VerifyResult = "mismatch"
)

type registryCache struct {
	FetchedAt time.Time `json:"fetchedAt"`
	Registry  Registry  `json:"registry"`
}

// RegistryCache fetches, caches, and queries the template registry.
type RegistryCache struct {
	cacheDir string
	cdnURL   string

	mu    sync.RWMutex
	cache *registryCache
}

func NewRegistryCache(cacheDir string) *RegistryCache {
	return &RegistryCache{
		cacheDir: cacheDir,
		cdnURL:   defaultRegistryURL,
	}
}

func (rc *RegistryCache) cachePath() string {
	return filepath.Join(rc.cacheDir, registryCacheFile)
}

// Load loads the registry from the local cache file, or fetches from CDN if
// the cache is missing or expired. Returns nil without error if unavailable.
func (rc *RegistryCache) Load() *Registry {
	rc.mu.RLock()
	if rc.cache != nil && time.Since(rc.cache.FetchedAt) < cacheTTL {
		reg := rc.cache.Registry
		rc.mu.RUnlock()
		return &reg
	}
	rc.mu.RUnlock()

	// Try loading from disk cache
	if data, err := os.ReadFile(rc.cachePath()); err == nil {
		var cached registryCache
		if err := json.Unmarshal(data, &cached); err == nil {
			rc.mu.Lock()
			rc.cache = &cached
			rc.mu.Unlock()
			if time.Since(cached.FetchedAt) < cacheTTL {
				return &cached.Registry
			}
		}
	}

	// Cache expired or missing — try to refresh from CDN
	if reg := rc.fetchFromCDN(); reg != nil {
		return reg
	}

	// CDN unreachable — return stale cache if available
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	if rc.cache != nil {
		reg := rc.cache.Registry
		return &reg
	}

	return nil
}

func (rc *RegistryCache) RefreshAsync() {
	go rc.fetchFromCDN()
}

func (rc *RegistryCache) fetchFromCDN() *Registry {
	// SEC-46: Validate redirects to prevent CDN hijacking
	client := &http.Client{
		Timeout: fetchTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			host := req.URL.Host
			if host != "presto.c-1o.top" && !isAllowedDownloadHost(host) {
				return fmt.Errorf("redirect to disallowed host: %s", host)
			}
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}
	resp, err := client.Get(rc.cdnURL)
	if err != nil {
		log.Printf("[registry] fetch failed: %v", err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[registry] fetch returned status %d", resp.StatusCode)
		return nil
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxRegistrySize))
	if err != nil {
		log.Printf("[registry] read body failed: %v", err)
		return nil
	}

	var reg Registry
	if err := json.Unmarshal(data, &reg); err != nil {
		log.Printf("[registry] parse failed: %v", err)
		return nil
	}

	cached := &registryCache{
		FetchedAt: time.Now(),
		Registry:  reg,
	}

	rc.mu.Lock()
	rc.cache = cached
	rc.mu.Unlock()

	if cacheData, err := json.Marshal(cached); err == nil {
		if err := os.WriteFile(rc.cachePath(), cacheData, 0600); err != nil { // SEC-45
			log.Printf("[registry] cache write failed: %v", err)
		}
	}

	log.Printf("[registry] refreshed, %d templates", len(reg.Templates))
	return &reg
}

func (rc *RegistryCache) VerifySHA256(templateName, actualSHA256 string) VerifyResult {
	reg := rc.Load()
	if reg == nil {
		return VerifyPending
	}

	platform := Platform()

	for _, entry := range reg.Templates {
		if entry.Name != templateName {
			continue
		}
		info, ok := entry.Platforms[platform]
		if !ok || info.SHA256 == "" {
			return VerifyNotInRegistry
		}
		if info.SHA256 == actualSHA256 {
			return VerifyMatched
		}
		return VerifyMismatch
	}

	return VerifyNotInRegistry
}

func (rc *RegistryCache) LookupTrust(templateName string) string {
	reg := rc.Load()
	if reg == nil {
		return ""
	}
	for _, entry := range reg.Templates {
		if entry.Name == templateName {
			return entry.Trust
		}
	}
	return ""
}

// SEC-39: LookupByRepo returns the registry entry for a template by owner/repo.
// The server uses this to get trusted download URLs instead of accepting client-provided URLs.
func (rc *RegistryCache) LookupByRepo(ownerRepo string) *RegistryEntry {
	reg := rc.Load()
	if reg == nil {
		return nil
	}
	for _, entry := range reg.Templates {
		if entry.Repo == ownerRepo {
			return &entry
		}
	}
	return nil
}

// LookupByName returns the registry entry for a template by its name.
// This is more reliable than LookupByRepo for monorepos that contain multiple templates.
func (rc *RegistryCache) LookupByName(name string) *RegistryEntry {
	reg := rc.Load()
	if reg == nil {
		return nil
	}
	for _, entry := range reg.Templates {
		if entry.Name == name {
			return &entry
		}
	}
	return nil
}

func Platform() string {
	return runtime.GOOS + "-" + runtime.GOARCH
}
