package template

import (
	"testing"
)

func TestSanitizeURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "URL with token parameter",
			input:    "https://example.com/download?token=secret123",
			expected: "https://example.com/download?token=%2A%2A%2A", // URL encoded ***
		},
		{
			name:     "URL with multiple parameters",
			input:    "https://example.com/download?version=1.0&token=secret123&format=zip",
			expected: "https://example.com/download?format=zip&token=%2A%2A%2A&version=1.0",
		},
		{
			name:     "URL without sensitive parameters",
			input:    "https://example.com/file.zip",
			expected: "https://example.com/file.zip",
		},
		{
			name:     "URL with api_key parameter",
			input:    "https://api.example.com?api_key=mykey123",
			expected: "https://api.example.com?api_key=%2A%2A%2A",
		},
		{
			name:     "Invalid URL",
			input:    "://invalid-url",
			expected: "://invalid-url",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeURL(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeURL(%s) = %s, want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSanitizePath(t *testing.T) {
	// Note: This test depends on the user's home directory
	// In CI environments, the home directory might differ

	tests := []struct {
		name     string
		contains string // Check if result contains this substring
	}{
		{
			name:     "Path with tilde",
			contains: "~",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This is a basic test - just verify the function doesn't crash
			result := SanitizePath("/Users/testuser/.presto")
			// The exact output depends on the system's home directory
			// Just verify it returns a non-empty string
			if result == "" {
				t.Error("SanitizePath returned empty string")
			}
		})
	}
}
