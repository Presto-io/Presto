package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/mrered/presto/internal/api"
	"github.com/mrered/presto/internal/appdata"
	"github.com/mrered/presto/internal/template"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// SEC-14: Default to localhost instead of all interfaces
	host := os.Getenv("HOST")
	if host == "" {
		host = "127.0.0.1"
	}

	dirs, err := appdata.ResolveDirs()
	if err != nil {
		log.Fatal("failed to resolve app data directories: ", err)
	}
	if result, err := appdata.MigrateLegacyOnce(dirs); err != nil {
		log.Printf("[presto] failed to migrate legacy app data: %v", err)
	} else if result.Attempted && len(result.Migrated) > 0 {
		log.Printf("[presto] migrated legacy app data: %s", strings.Join(result.Migrated, ","))
	}
	if err := dirs.Ensure(); err != nil {
		log.Fatal("failed to create app data directories: ", err)
	}
	prestoDir := dirs.DataDir
	templatesDir := dirs.TemplatesDir()
	if err := os.MkdirAll(templatesDir, 0755); err != nil {
		log.Fatal("failed to create templates directory: ", err)
	}

	staticDir := os.Getenv("STATIC_DIR")
	if staticDir == "" {
		staticDir = "frontend/build"
	}

	// SEC-09: API key authentication
	apiKey := os.Getenv("PRESTO_API_KEY")
	if apiKey == "" {
		b := make([]byte, 32)
		if _, err := rand.Read(b); err != nil {
			log.Fatal("failed to generate API key: ", err)
		}
		apiKey = hex.EncodeToString(b)
	}

	// Font paths: default to the Presto data dir; FONT_PATHS can add more directories.
	// SEC-44: Check MkdirAll error
	fontsDir := dirs.FontsDir()
	if err := os.MkdirAll(fontsDir, 0755); err != nil {
		log.Fatal("failed to create fonts directory: ", err)
	}
	if err := appdata.MarkGenerated(prestoDir); err != nil {
		log.Printf("[presto] failed to mark generated app data: %v", err)
	}
	fontPaths := appdata.ResolveFontPaths(fontsDir)

	// Registry cache for SHA256 verification of imported templates
	registry := template.NewRegistryCache(dirs.CacheDir)
	registry.RefreshAsync()
	manager := template.NewManager(templatesDir)
	go installOfficialTemplatesOnStartup(manager, registry)

	srv := api.NewServer(api.ServerOptions{
		TemplatesDir: templatesDir,
		StaticDir:    staticDir,
		TypstBin:     "typst",
		APIKey:       apiKey,
		InjectAPIKey: shouldInjectAPIKey(host),
		FontPaths:    fontPaths,
		Registry:     registry,
	})

	addr := fmt.Sprintf("%s:%s", host, port)
	fmt.Printf("Presto server listening on %s\n", addr)
	// SEC-43: Only show truncated API key to avoid full key in logs
	fmt.Printf("API Key: %s...%s\n", apiKey[:8], apiKey[len(apiKey)-4:])
	server := &http.Server{
		Addr:              addr,
		Handler:           srv,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      2 * time.Minute,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
	log.Fatal(server.ListenAndServe())
}

func shouldInjectAPIKey(host string) bool {
	if override := os.Getenv("PRESTO_INJECT_API_KEY"); override != "" {
		return override == "1" || strings.EqualFold(override, "true") || strings.EqualFold(override, "yes")
	}
	return host == "127.0.0.1" || host == "localhost" || host == "::1"
}
