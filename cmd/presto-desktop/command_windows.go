//go:build windows

package main

import (
	"os/exec"
	"syscall"
)

const createNoWindow = 0x08000000

func execCommand(name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: createNoWindow}
	return cmd
}
