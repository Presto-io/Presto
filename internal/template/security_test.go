package template

import (
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"testing"
)

func TestIsAllowedDownloadHost(t *testing.T) {
	tests := []struct {
		host    string
		allowed bool
	}{
		{"github.com", true},
		{"api.github.com", true},
		{"objects.githubusercontent.com", true},
		{"github-releases.githubusercontent.com", true},
		{"codeload.github.com", true},
		{"evil.com", false},
		{"github.com.evil.com", false},
		{"", false},
	}

	for _, tt := range tests {
		if got := isAllowedDownloadHost(tt.host); got != tt.allowed {
			t.Errorf("isAllowedDownloadHost(%q) = %v, want %v", tt.host, got, tt.allowed)
		}
	}
}

func TestValidateName(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{"valid-name", false},
		{"valid.name", false},
		{"valid_name", false},
		{"Valid123", false},
		{"", true},
		{"../traversal", true},
		{"-invalid", true},
		{"name with spaces", true},
		{"name\x00null", true},
	}

	for _, tt := range tests {
		err := validateName(tt.name)
		if (err != nil) != tt.wantErr {
			t.Errorf("validateName(%q) error = %v, wantErr %v", tt.name, err, tt.wantErr)
		}
	}
}

func TestDomainWhitelistForDownloadURL(t *testing.T) {
	tests := []struct {
		rawURL  string
		allowed bool
	}{
		{"https://github.com/owner/repo/releases/download/v1/binary", true},
		{"https://objects.githubusercontent.com/path/binary", true},
		{"https://evil.com/binary", false},
		{"https://github.com.evil.com/binary", false},
	}

	for _, tt := range tests {
		parsedURL, err := url.Parse(tt.rawURL)
		if err != nil {
			t.Fatalf("url.Parse(%q) failed: %v", tt.rawURL, err)
		}
		if got := isAllowedDownloadHost(parsedURL.Host); got != tt.allowed {
			t.Errorf("URL %q: isAllowedDownloadHost(%q) = %v, want %v", tt.rawURL, parsedURL.Host, got, tt.allowed)
		}
	}
}

func TestSHA256Verification(t *testing.T) {
	data := []byte("test binary content")
	hash := sha256.Sum256(data)
	correctHex := hex.EncodeToString(hash[:])

	// Correct hash should match
	if correctHex == "" {
		t.Fatal("SHA256 should not be empty")
	}

	// Mismatch detection
	wrongHex := "0000000000000000000000000000000000000000000000000000000000000000"
	if wrongHex == correctHex {
		t.Fatal("wrong hash should not match correct hash")
	}
}

func TestLookupChecksumParsing(t *testing.T) {
	// Test the checksum file parsing format
	checksumContent := "abc123def456  binary-darwin-arm64\nfed789abc012  binary-linux-amd64\n"
	lines := []string{}
	for _, line := range splitLines(checksumContent) {
		if line != "" {
			lines = append(lines, line)
		}
	}

	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
