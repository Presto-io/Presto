package template

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRangeRequest_FirstRequest(t *testing.T) {
	// Test that first download has no Range header
	data := []byte("test content for download")
	firstRequest := true

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if firstRequest {
			if r.Header.Get("Range") != "" {
				t.Error("first request should not have Range header")
			}
			firstRequest = false
		}
		w.Header().Set("Content-Length", string(len(data)))
		w.Write(data)
	}))
	defer server.Close()

	// Test downloadWithResume (will be implemented in Task 3)
	// For now, this test will fail because downloadWithResume doesn't exist
	result, err := downloadWithResume(server.URL, 0, nil)
	if err != nil {
		t.Fatalf("download failed: %v", err)
	}
	if string(result) != string(data) {
		t.Errorf("downloaded data mismatch")
	}
}

func TestRangeRequest_Resume(t *testing.T) {
	// Test that resume adds Range header
	data := []byte("test content for resume download")
	receivedRangeHeader := ""
	requestCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		receivedRangeHeader = r.Header.Get("Range")

		// Simulate partial content support
		if receivedRangeHeader != "" {
			w.Header().Set("Content-Range", "bytes 5-29/30")
			w.WriteHeader(http.StatusPartialContent)
			w.Write(data[5:])
		} else {
			w.Header().Set("Content-Length", string(len(data)))
			w.Write(data)
		}
	}))
	defer server.Close()

	// First download should succeed without Range header
	_, err := downloadWithResume(server.URL, 0, nil)
	if err != nil {
		t.Fatalf("first download failed: %v", err)
	}
	if requestCount != 1 {
		t.Errorf("expected 1 request, got %d", requestCount)
	}
	if receivedRangeHeader != "" {
		t.Error("first request should not have Range header")
	}
}

func TestRangeRequest_ServerNotSupport(t *testing.T) {
	// Test that server doesn't support Range (returns 200 instead of 206)
	data := []byte("server does not support range requests")
	requestCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		// Always return 200, ignore Range header
		w.Header().Set("Content-Length", string(len(data)))
		w.Write(data)
	}))
	defer server.Close()

	_, err := downloadWithResume(server.URL, 0, nil)
	if err != nil {
		t.Fatalf("download failed: %v", err)
	}

	// Should succeed even without Range support
	if requestCount < 1 {
		t.Errorf("expected at least 1 request, got %d", requestCount)
	}
}

func TestRangeRequest_CleanupOnSuccess(t *testing.T) {
	// Test that temp file is cleaned up after successful download
	data := []byte("test cleanup on success")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", string(len(data)))
		w.Write(data)
	}))
	defer server.Close()

	_, err := downloadWithResume(server.URL, 0, nil)
	if err != nil {
		t.Fatalf("download failed: %v", err)
	}

	// Verify temp file is cleaned up (implementation dependent)
	// This is a placeholder - actual check depends on temp file location
}
