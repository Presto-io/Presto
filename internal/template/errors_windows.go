// +build windows

package template

import (
	"fmt"
	"syscall"
)

// windowsErrorMessage maps Windows error codes to user-friendly messages
func windowsErrorMessage(errno syscall.Errno) string {
	messages := map[syscall.Errno]string{
		syscall.ERROR_ACCESS_DENIED:          "权限被拒绝（需要管理员权限或文件被占用）",
		syscall.ERROR_PATH_NOT_FOUND:         "路径不存在",
		syscall.ERROR_FILE_NOT_FOUND:         "文件不存在",
		syscall.ERROR_SHARING_VIOLATION:      "文件正在被其他程序使用",
		syscall.ERROR_LOCK_VIOLATION:         "文件被锁定",
		syscall.ERROR_NETWORK_BUSY:           "网络繁忙",
		syscall.ERROR_NETWORK_UNREACHABLE:    "网络不可达",
		syscall.ERROR_CONNECTION_REFUSED:     "连接被拒绝",
		syscall.ERROR_CONNECTION_ABORTED:     "连接中断",
		syscall.ERROR_CONNECTION_RESET:       "连接重置",
		syscall.ERROR_NETNAME_DELETED:        "网络名称已删除",
		syscall.WSAECONNREFUSED:              "连接被拒绝（防火墙可能阻止连接）",
		syscall.WSAENETUNREACH:               "网络不可达",
		syscall.WSAETIMEDOUT:                 "连接超时",
	}

	if msg, ok := messages[errno]; ok {
		return msg
	}
	return fmt.Sprintf("Windows 错误码: %d", errno)
}
