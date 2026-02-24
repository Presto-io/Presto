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

	home, _ := os.UserHomeDir()
	prestoDir := filepath.Join(home, ".presto")
	templatesDir := filepath.Join(prestoDir, "templates")
	os.MkdirAll(templatesDir, 0755)

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
	fontsDir := filepath.Join(prestoDir, "fonts")
	os.MkdirAll(fontsDir, 0755)
	var fontPaths []string
	if fp := os.Getenv("FONT_PATHS"); fp != "" {
		fontPaths = strings.Split(fp, ":")
	} else {
		fontPaths = []string{fontsDir}
	}

	// Registry cache for SHA256 verification of imported templates
	registry := template.NewRegistryCache(prestoDir)
	registry.RefreshAsync()

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
	fmt.Printf("API Key: %s\n", apiKey)
	log.Fatal(http.ListenAndServe(addr, srv))
}
