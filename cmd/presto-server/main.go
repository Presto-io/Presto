package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

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
	templatesDir := filepath.Join(home, ".presto", "templates")
	os.MkdirAll(templatesDir, 0755)

	// Auto-install bundled official templates if missing
	mgr := template.NewManager(templatesDir)
	exePath, _ := os.Executable()
	if exePath != "" {
		bundleDir := filepath.Join(filepath.Dir(exePath), "templates")
		mgr.EnsureOfficialTemplates(bundleDir)
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

	srv := api.NewServer(api.ServerOptions{
		TemplatesDir: templatesDir,
		StaticDir:    staticDir,
		TypstBin:     "typst",
		APIKey:       apiKey,
	})

	addr := fmt.Sprintf("%s:%s", host, port)
	fmt.Printf("Presto server listening on %s\n", addr)
	fmt.Printf("API Key: %s\n", apiKey)
	log.Fatal(http.ListenAndServe(addr, srv))
}
