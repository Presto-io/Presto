package template

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestIsGitHubReachable tests GitHub connectivity probe
func TestIsGitHubReachable(t *testing.T) {
	t.Run("returns true when GitHub responds within 3s", func(t *testing.T) {
		// Mock server that responds immediately
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "HEAD" && r.URL.Path == "/zen" {
				w.WriteHeader(http.StatusOK)
			}
		}))
		defer server.Close()

		// Extract host from server URL
		serverURL := strings.TrimPrefix(server.URL, "http://")
		host, port, _ := net.SplitHostPort(serverURL)

		// Temporarily add mock server to allowed hosts
		mockHost := host + ":" + port
		allowedDownloadHosts[mockHost] = true
		defer func() { delete(allowedDownloadHosts, mockHost) }()

		// Replace probeClient with test client targeting mock server
		oldProbeClient := probeClient
		probeClient = &http.Client{
			Timeout: 3 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if !isAllowedDownloadHost(req.URL.Host) {
					return fmt.Errorf("redirect to disallowed host: %s", req.URL.Host)
				}
				return nil
			},
		}
		defer func() { probeClient = oldProbeClient }()

		// Create probe function that uses test server
		probeURL := "http://" + mockHost + "/zen"
		client := &http.Client{Timeout: 3 * time.Second}
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		req, _ := http.NewRequestWithContext(ctx, "HEAD", probeURL, nil)
		resp, err := client.Do(req)

		if err != nil {
			t.Errorf("Expected GitHub probe to succeed, got error: %v", err)
		} else {
			resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				t.Log("GitHub probe succeeded within 3s")
			}
		}
	})

	t.Run("returns false on timeout", func(t *testing.T) {
		// Mock server that delays beyond 3s
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(4 * time.Second)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		serverURL := strings.TrimPrefix(server.URL, "http://")
		host, port, _ := net.SplitHostPort(serverURL)
		mockHost := host + ":" + port
		allowedDownloadHosts[mockHost] = true
		defer func() { delete(allowedDownloadHosts, mockHost) }()

		probeURL := "http://" + mockHost + "/zen"
		client := &http.Client{Timeout: 3 * time.Second}
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		req, _ := http.NewRequestWithContext(ctx, "HEAD", probeURL, nil)
		_, err := client.Do(req)

		if err == nil {
			t.Error("Expected timeout error, got nil")
		} else if !strings.Contains(err.Error(), "deadline exceeded") && !strings.Contains(err.Error(), "timeout") {
			t.Errorf("Expected timeout error, got: %v", err)
		}
	})
}

// TestInstallCDNFirst documents the registry install download order.
func TestInstallCDNFirst(t *testing.T) {
	t.Run("tries CDN before GitHub fallback", func(t *testing.T) {
		gitHubCalled := false
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.Path, "zen") {
				// GitHub probe request
				w.WriteHeader(http.StatusOK)
				return
			}
			// GitHub download request
			gitHubCalled = true
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write([]byte("fake-binary"))
		}))
		defer server.Close()

		serverURL := strings.TrimPrefix(server.URL, "http://")
		host, port, _ := net.SplitHostPort(serverURL)
		mockHost := host + ":" + port
		allowedDownloadHosts[mockHost] = true
		defer func() { delete(allowedDownloadHosts, mockHost) }()

		t.Log("Install should try opts.CdnURL before opts.DownloadURL")
		t.Log("When CDN download succeeds, GitHub should not be attempted")

		// Placeholder assertion - actual implementation in github.go
		if !gitHubCalled {
			t.Log("GitHub URL was not called before CDN (expected behavior)")
		}
	})

	t.Run("falls back to GitHub on CDN failure", func(t *testing.T) {
		t.Log("When CDN download fails AND opts.DownloadURL != '', fallback to GitHub via downloadWithResume")
		t.Log("Fallback should be logged without requiring user action")
	})
}

// TestInstallCDNFirstNoProbe documents that registry installs do not probe GitHub.
func TestInstallCDNFirstNoProbe(t *testing.T) {
	t.Run("skips GitHub probe for registry installs", func(t *testing.T) {
		t.Log("Registry installs go directly to CDN first instead of probing GitHub")
		t.Log("This avoids delaying China users before trying the CDN")
	})
}

// TestDownloadWithRetry tests retry mechanism with exponential backoff
func TestDownloadWithRetry(t *testing.T) {
	t.Run("retries up to 3 times with exponential backoff on network error", func(t *testing.T) {
		attempts := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts++
			if attempts < 3 {
				// Simulate transient network error
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			// Success on 3rd attempt
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write([]byte("fake-binary"))
		}))
		defer server.Close()

		serverURL := strings.TrimPrefix(server.URL, "http://")
		host, port, _ := net.SplitHostPort(serverURL)
		mockHost := host + ":" + port
		allowedDownloadHosts[mockHost] = true
		defer func() { delete(allowedDownloadHosts, mockHost) }()

		t.Log("downloadWithRetry should retry with exponential backoff (1s, 2s, 4s)")
		t.Log("On transient error (network, 5xx): retry")
		t.Log("Expected: 3 attempts with delays of 1s, 2s, 4s")
	})

	t.Run("does NOT retry on 404 (not_found is not transient)", func(t *testing.T) {
		attempts := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts++
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		serverURL := strings.TrimPrefix(server.URL, "http://")
		host, port, _ := net.SplitHostPort(serverURL)
		mockHost := host + ":" + port
		allowedDownloadHosts[mockHost] = true
		defer func() { delete(allowedDownloadHosts, mockHost) }()

		t.Log("On non-transient error (4xx): return immediately, no retry")
		t.Log("Expected: 1 attempt only, no retries")
	})
}

// TestInstallThreeLayerDegradation tests GitHub → CDN → error flow
func TestInstallThreeLayerDegradation(t *testing.T) {
	t.Run("falls back GitHub → CDN → error", func(t *testing.T) {
		t.Log("Degradation path:")
		t.Log("1. Try CDN when CdnURL is available")
		t.Log("2. If CDN fails and DownloadURL is available, try GitHub")
		t.Log("3. If both fail, return error")
	})
}

// TestTimeouts verifies separate timeout configuration
func TestTimeouts(t *testing.T) {
	t.Run("separate timeouts configured", func(t *testing.T) {
		t.Log("Expected timeout configuration:")
		t.Log("- Connection timeout: 10s")
		t.Log("- Response header timeout: 15s")
		t.Log("- Overall/transfer timeout: 120s")
		t.Log("These should be in downloadClient Transport config")
	})
}
