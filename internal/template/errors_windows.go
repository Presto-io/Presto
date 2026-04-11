// +build windows

package template

import (
	"fmt"
	"syscall"
)

const (
	windowsErrFileNotFound     syscall.Errno = 2
	windowsErrPathNotFound     syscall.Errno = 3
	windowsErrAccessDenied     syscall.Errno = 5
	windowsErrSharingViolation syscall.Errno = 32
	windowsErrLockViolation    syscall.Errno = 33
	windowsErrNetworkBusy      syscall.Errno = 54
	windowsErrNetNameDeleted   syscall.Errno = 64
	winsockErrConnRefused      syscall.Errno = 10061
	winsockErrNetUnreachable   syscall.Errno = 10051
	winsockErrTimedOut         syscall.Errno = 10060
)

// windowsErrorMessage maps Windows error codes to user-friendly messages
func windowsErrorMessage(errno syscall.Errno) string {
	switch errno {
	case windowsErrAccessDenied:
		return "权限被拒绝（需要管理员权限或文件被占用）"
	case windowsErrPathNotFound:
		return "路径不存在"
	case windowsErrFileNotFound:
		return "文件不存在"
	case windowsErrSharingViolation:
		return "文件正在被其他程序使用"
	case windowsErrLockViolation:
		return "文件被锁定"
	case windowsErrNetworkBusy:
		return "网络繁忙"
	case windowsErrNetNameDeleted:
		return "网络名称已删除"
	case winsockErrConnRefused:
		return "连接被拒绝（防火墙可能阻止连接）"
	case winsockErrNetUnreachable:
		return "网络不可达"
	case winsockErrTimedOut:
		return "连接超时"
	}

	if message := errno.Error(); message != "" {
		return fmt.Sprintf("Windows 错误: %s (%d)", message, errno)
	}
	return fmt.Sprintf("Windows 错误码: %d", errno)
}
