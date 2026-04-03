// +build !windows

package template

import (
	"os"
)

// validateWindowsPath is a no-op on non-Windows platforms
func validateWindowsPath(path string) error {
	return nil
}

// isWindowsPermissionError is a no-op on non-Windows platforms
func isWindowsPermissionError(err error) bool {
	return false
}

// windowsPermissionErrorMsg returns empty string on non-Windows platforms
func windowsPermissionErrorMsg(path string) string {
	return ""
}

// Dummy reference to os to satisfy imports
var _ = os.PathError{}
