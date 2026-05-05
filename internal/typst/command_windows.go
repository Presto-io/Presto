//go:build windows

package typst

import (
	"context"
	"os/exec"
	"syscall"
)

const createNoWindow = 0x08000000

func execCommandContext(ctx context.Context, name string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: createNoWindow}
	return cmd
}
