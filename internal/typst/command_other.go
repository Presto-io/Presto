//go:build !windows

package typst

import (
	"context"
	"os/exec"
)

func execCommandContext(ctx context.Context, name string, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, name, args...)
}
