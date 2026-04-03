// +build windows

package template

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

const (
	// MAX_PATH is the maximum path length on Windows (260 characters)
	MAX_PATH = 260
)

// validateWindowsPath checks for Windows-specific path issues
func validateWindowsPath(path string) error {
	// Check path length
	if len(path) > MAX_PATH {
		return fmt.Errorf("path too long (%d > %d): %s", len(path), MAX_PATH, path)
	}

	// Check if path contains invalid characters
	invalidChars := `<>:"|?*`
	for _, ch := range invalidChars {
		if containsRune(path, ch) {
			return fmt.Errorf("path contains invalid character '%c': %s", ch, path)
		}
	}

	// Check if path is a reserved name (CON, PRN, AUX, NUL, COM1-9, LPT1-9)
	base := filepath.Base(path)
	reserved := []string{"CON", "PRN", "AUX", "NUL"}
	for i := 1; i <= 9; i++ {
		reserved = append(reserved, fmt.Sprintf("COM%d", i), fmt.Sprintf("LPT%d", i))
	}
	for _, r := range reserved {
		if base == r || base == r+"." {
			return fmt.Errorf("path uses reserved name '%s': %s", r, path)
		}
	}

	return nil
}

func containsRune(s string, r rune) bool {
	for _, ch := range s {
		if ch == r {
			return true
		}
	}
	return false
}

// isWindowsPermissionError checks if an error is a Windows permission error
func isWindowsPermissionError(err error) bool {
	if pathErr, ok := err.(*os.PathError); ok {
		if errno, ok := pathErr.Err.(syscall.Errno); ok {
			return errno == syscall.ERROR_ACCESS_DENIED
		}
	}
	return false
}

// windowsPermissionErrorMsg returns a user-friendly permission error message
func windowsPermissionErrorMsg(path string) string {
	return fmt.Sprintf("无法写入文件或目录（权限被拒绝）:\n\n"+
		"路径: %s\n\n"+
		"可能的原因:\n"+
		"1. 杀毒软件正在扫描该文件（请稍后重试）\n"+
		"2. 文件被其他程序占用（关闭其他程序）\n"+
		"3. 需要管理员权限（右键 Presto → 以管理员身份运行）\n\n"+
		"建议: 暂时禁用杀毒软件或以管理员身份运行 Presto", path)
}
