package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/mrered/presto/internal/api"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := api.NewServer()
	fmt.Printf("Presto server listening on :%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, srv))
}
