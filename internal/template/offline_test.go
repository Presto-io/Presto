package template

import (
	"testing"
)

func TestOfflineCache_ListInstalled(t *testing.T) {
	// Test that List() returns installed templates even when offline
	// This is a placeholder - actual implementation depends on manager.List()

	// Placeholder: verify the function exists
	// Real test would:
	// 1. Create a mock manager with some installed templates
	// 2. Call List()
	// 3. Verify it returns templates without network request

	t.Log("placeholder - list installed templates without network")
}

func TestOfflineCache_NoNetwork(t *testing.T) {
	// Test that offline detection prevents network requests
	// This verifies that navigator.onLine is checked before attempting downloads

	// Placeholder: verify offline check logic
	// Real test would:
	// 1. Set network state to offline
	// 2. Attempt to list templates from registry
	// 3. Verify no network request is made
	// 4. Verify local cache is used instead

	t.Log("placeholder - no network requests when offline")
}
