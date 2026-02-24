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

// RegistryPlatformInfo holds per-platform binary URL and SHA256.
type RegistryPlatformInfo struct {
	URL    string `json:"url"`
	SHA256 string `json:"sha256"`
}

// RegistryEntry represents a single template in the registry.
type RegistryEntry struct {
	Name      string                          `json:"name"`
	Version   string                          `json:"version"`
	Trust     string                          `json:"trust"`
	Repo      string                          `json:"repo"`
	Platforms map[string]RegistryPlatformInfo `json:"platforms"`
}

// Registry is the top-level registry structure.
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

// registryCache wraps cached file metadata.
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

// NewRegistryCache creates a new RegistryCache.
func NewRegistryCache(cacheDir string) *RegistryCache {
	return &RegistryCache{
		cacheDir: cacheDir,
		cdnURL:   defaultRegistryURL,
	}
}

// cachePath returns the full path to the cache file.
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

// RefreshAsync refreshes the registry cache in the background.
func (rc *RegistryCache) RefreshAsync() {
	go func() {
		rc.fetchFromCDN()
	}()
}

// fetchFromCDN downloads the registry from the CDN and updates the cache.
func (rc *RegistryCache) fetchFromCDN() *Registry {
	client := &http.Client{Timeout: fetchTimeout}
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

	// Persist to disk
	if cacheData, err := json.Marshal(cached); err == nil {
		if err := os.WriteFile(rc.cachePath(), cacheData, 0644); err != nil {
			log.Printf("[registry] cache write failed: %v", err)
		}
	}

	log.Printf("[registry] refreshed, %d templates", len(reg.Templates))
	return &reg
}

// VerifySHA256 checks a binary's SHA256 against the registry for the given
// template name and current platform. Returns the verification result.
func (rc *RegistryCache) VerifySHA256(templateName, actualSHA256 string) VerifyResult {
	reg := rc.Load()
	if reg == nil {
		return VerifyPending
	}

	platform := runtime.GOOS + "-" + runtime.GOARCH

	for _, entry := range reg.Templates {
		if entry.Name != templateName {
			continue
		}
		info, ok := entry.Platforms[platform]
		if !ok || info.SHA256 == "" {
			// Template is in registry but no hash for this platform
			return VerifyNotInRegistry
		}
		if info.SHA256 == actualSHA256 {
			return VerifyMatched
		}
		return VerifyMismatch
	}

	return VerifyNotInRegistry
}

// LookupTrust returns the trust level for a template from the registry.
// Returns empty string if not found.
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

// Platform returns the current platform string (e.g. "darwin-arm64").
func Platform() string {
	return fmt.Sprintf("%s-%s", runtime.GOOS, runtime.GOARCH)
}
