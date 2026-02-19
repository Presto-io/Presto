package template

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// SEC-12: Timeout for template binary execution
const executorTimeout = 30 * time.Second

type Executor struct {
	BinaryPath string
}

func NewExecutor(binaryPath string) *Executor {
	return &Executor{BinaryPath: binaryPath}
}

// SEC-10: Minimal environment for template execution
// SEC-12: Timeout via context
func (e *Executor) Convert(markdown string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), executorTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, e.BinaryPath)
	cmd.Env = []string{"PATH=/usr/local/bin:/usr/bin:/bin"}
	cmd.Stdin = strings.NewReader(markdown)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("template execution timed out after %s", executorTimeout)
		}
		return "", fmt.Errorf("template execution failed: %w\nstderr: %s", err, stderr.String())
	}
	return stdout.String(), nil
}

func (e *Executor) GetManifest() ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), executorTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, e.BinaryPath, "--manifest")
	cmd.Env = []string{"PATH=/usr/local/bin:/usr/bin:/bin"}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("manifest retrieval timed out after %s", executorTimeout)
		}
		return nil, fmt.Errorf("manifest retrieval failed: %w\nstderr: %s", err, stderr.String())
	}
	return stdout.Bytes(), nil
}

func (e *Executor) GetExample() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), executorTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, e.BinaryPath, "--example")
	cmd.Env = []string{"PATH=/usr/local/bin:/usr/bin:/bin"}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("example retrieval timed out after %s", executorTimeout)
		}
		return "", fmt.Errorf("example retrieval failed: %w\nstderr: %s", err, stderr.String())
	}
	return stdout.String(), nil
}
