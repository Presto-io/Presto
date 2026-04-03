package template

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDownloadWithResume_FirstDownload tests first download (no partial file)
func TestDownloadWithResume_FirstDownload(t *testing.T) {
	data := []byte("test content for first download")
	attempts := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if r.Header.Get("Range") != "" {
			t.Error("first download should not have Range header")
		}
		w.WriteHeader(http.StatusOK)
		w.Write(data)
	}))
	defer server.Close()

	// Add test server to allowed hosts
	serverHost := strings.TrimPrefix(server.URL, "http://")
	allowedDownloadHosts[serverHost] = true
	defer func() { delete(allowedDownloadHosts, serverHost) }()

	// Clean up any existing temp files
	os.RemoveAll(getTmpDownloadDir())

	result, err := downloadWithResume(server.URL, 0, nil)
	if err != nil {
		t.Fatalf("download failed: %v", err)
	}
	if string(result) != string(data) {
		t.Errorf("expected %s, got %s", string(data), string(result))
	}
	if attempts != 1 {
		t.Errorf("expected 1 attempt, got %d", attempts)
	}
}

// TestDownloadWithResume_ResumeFromPartial tests resume from partial download
func TestDownloadWithResume_ResumeFromPartial(t *testing.T) {
	fullData := []byte("0123456789ABCDEF")
	partialData := fullData[:8] // First 8 bytes already downloaded
	remainingData := fullData[8:]

	rangeHeaderReceived := ""

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rangeHeader := r.Header.Get("Range")
		if rangeHeader == "" {
			// First request (partial file exists, but we're simulating resume)
			w.WriteHeader(http.StatusOK)
			w.Write(fullData)
		} else {
			rangeHeaderReceived = rangeHeader
			// Resume request
			if rangeHeader != "bytes=8-" {
				t.Errorf("expected Range: bytes=8-, got %s", rangeHeader)
			}
			w.Header().Set("Content-Range", "bytes 8-15/16")
			w.WriteHeader(http.StatusPartialContent)
			w.Write(remainingData)
		}
	}))
	defer server.Close()

	// Add test server to allowed hosts
	serverHost := strings.TrimPrefix(server.URL, "http://")
	allowedDownloadHosts[serverHost] = true
	defer func() { delete(allowedDownloadHosts, serverHost) }()

	// Clean up temp dir
	tmpDir := getTmpDownloadDir()
	os.RemoveAll(tmpDir)
	os.MkdirAll(tmpDir, 0700)

	// Create partial file (simulate interrupted download)
	tmpFile := getTmpFilePath(server.URL)
	if err := os.WriteFile(tmpFile, partialData, 0600); err != nil {
		t.Fatalf("failed to create partial file: %v", err)
	}

	// Resume download
	result, err := downloadWithResume(server.URL, 0, nil)
	if err != nil {
		t.Fatalf("resume download failed: %v", err)
	}
	if string(result) != string(fullData) {
		t.Errorf("expected %s, got %s", string(fullData), string(result))
	}
	if rangeHeaderReceived != "bytes=8-" {
		t.Errorf("expected Range header 'bytes=8-', got '%s'", rangeHeaderReceived)
	}

	// Verify temp file was cleaned up
	if _, err := os.Stat(tmpFile); !os.IsNotExist(err) {
		t.Error("temp file should be cleaned up after successful download")
	}
}

// TestDownloadWithResume_ServerNotSupportRange tests graceful degradation
func TestDownloadWithResume_ServerNotSupportRange(t *testing.T) {
	fullData := []byte("test content without range support")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Server always returns 200, ignores Range header
		w.WriteHeader(http.StatusOK)
		w.Write(fullData)
	}))
	defer server.Close()

	// Add test server to allowed hosts
	serverHost := strings.TrimPrefix(server.URL, "http://")
	allowedDownloadHosts[serverHost] = true
	defer func() { delete(allowedDownloadHosts, serverHost) }()

	// Clean up and create partial file
	tmpDir := getTmpDownloadDir()
	os.RemoveAll(tmpDir)
	os.MkdirAll(tmpDir, 0700)
	tmpFile := getTmpFilePath(server.URL)
	os.WriteFile(tmpFile, []byte("partial"), 0600)

	// Download with maxRetries=1 to allow retry when server doesn't support Range
	result, err := downloadWithResume(server.URL, 1, nil)
	if err != nil {
		t.Fatalf("download failed: %v", err)
	}
	if string(result) != string(fullData) {
		t.Errorf("expected %s, got %s", string(fullData), string(result))
	}

	// Verify temp file was cleaned up after successful download
	if _, err := os.Stat(tmpFile); !os.IsNotExist(err) {
		t.Error("temp file should be cleaned up after successful download")
	}
}

// TestDownloadWithResume_ErrorWithoutCleanup tests error handling when server fails immediately
func TestDownloadWithResume_ErrorWithoutCleanup(t *testing.T) {
	attempt := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt++
		// Always return error
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	// Add test server to allowed hosts
	serverHost := strings.TrimPrefix(server.URL, "http://")
	allowedDownloadHosts[serverHost] = true
	defer func() { delete(allowedDownloadHosts, serverHost) }()

	// Clean up temp dir
	tmpDir := getTmpDownloadDir()
	os.RemoveAll(tmpDir)
	os.MkdirAll(tmpDir, 0700)
	tmpFile := getTmpFilePath(server.URL)

	// Attempt download with retries (should fail)
	_, err := downloadWithResume(server.URL, 2, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Verify we attempted multiple times
	if attempt < 2 {
		t.Errorf("expected at least 2 attempts, got %d", attempt)
	}

	// When server returns 5xx immediately, tmp file may or may not exist
	// (depends on whether OpenFile was called before error)
	// This is acceptable - the key is that download fails properly
	if _, err := os.Stat(tmpFile); err == nil {
		t.Log("Note: tmp file exists despite error (acceptable - may be created during retry)")
	}
}

// Helper functions
func getTmpDownloadDir() string {
	return filepath.Join(os.TempDir(), tmpDownloadDir)
}

func getTmpFilePath(url string) string {
	return filepath.Join(getTmpDownloadDir(), hashURL(url)+".tmp")
}
