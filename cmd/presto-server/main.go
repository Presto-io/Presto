package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/mrered/presto/internal/api"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	home, _ := os.UserHomeDir()
	templatesDir := filepath.Join(home, ".presto", "templates")
	os.MkdirAll(templatesDir, 0755)

	staticDir := os.Getenv("STATIC_DIR")
	if staticDir == "" {
		staticDir = "frontend/build"
	}

	srv := api.NewServer(templatesDir, staticDir)
	fmt.Printf("Presto server listening on :%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, srv))
}
