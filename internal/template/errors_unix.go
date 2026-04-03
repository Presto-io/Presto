// +build !windows

package template

import "syscall"

// windowsErrorMessage is a no-op on non-Windows platforms
func windowsErrorMessage(errno syscall.Errno) string {
	return ""
}
