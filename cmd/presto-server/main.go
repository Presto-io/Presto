package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/mrered/presto/internal/api"
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

	// SEC-44: Check os.UserHomeDir error
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal("failed to get home directory: ", err)
	}
	prestoDir := filepath.Join(home, ".presto")
	templatesDir := filepath.Join(prestoDir, "templates")
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

	// Font paths: default to ~/.presto/fonts, can override with FONT_PATHS (colon-separated)
	// SEC-44: Check MkdirAll error
	fontsDir := filepath.Join(prestoDir, "fonts")
	if err := os.MkdirAll(fontsDir, 0755); err != nil {
		log.Fatal("failed to create fonts directory: ", err)
	}
	var fontPaths []string
	if fp := os.Getenv("FONT_PATHS"); fp != "" {
		fontPaths = strings.Split(fp, ":")
	} else {
		fontPaths = []string{fontsDir}
	}

	// Registry cache for SHA256 verification of imported templates
	registry := template.NewRegistryCache(prestoDir)
	registry.RefreshAsync()
	manager := template.NewManager(templatesDir)
	go installOfficialTemplatesOnStartup(manager, registry)

	srv := api.NewServer(api.ServerOptions{
		TemplatesDir: templatesDir,
		StaticDir:    staticDir,
		TypstBin:     "typst",
		APIKey:       apiKey,
		FontPaths:    fontPaths,
		Registry:     registry,
	})

	addr := fmt.Sprintf("%s:%s", host, port)
	fmt.Printf("Presto server listening on %s\n", addr)
	// SEC-43: Only show truncated API key to avoid full key in logs
	fmt.Printf("API Key: %s...%s\n", apiKey[:8], apiKey[len(apiKey)-4:])
	log.Fatal(http.ListenAndServe(addr, srv))
}
