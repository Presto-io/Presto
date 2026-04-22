package template

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
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
func (e *Executor) run(args []string, stdin string) ([]byte, error) {
	var pathEnv string
	if runtime.GOOS == "windows" {
		pathEnv = os.TempDir() + ";C:\\Windows\\System32"
	} else {
		pathEnv = "/usr/local/bin:/usr/bin:/bin"
	}

	ctx, cancel := context.WithTimeout(context.Background(), executorTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, e.BinaryPath, args...)
	cmd.Env = []string{"PATH=" + pathEnv}
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("execution timed out after %s", executorTimeout)
		}
		return nil, fmt.Errorf("execution failed: %w\nstderr: %s", err, stderr.String())
	}
	return stdout.Bytes(), nil
}

func (e *Executor) Convert(markdown string) (string, error) {
	out, err := e.run(nil, markdown)
	if err != nil {
		return "", fmt.Errorf("template convert: %w", err)
	}
	return string(out), nil
}

func (e *Executor) GetManifest() ([]byte, error) {
	out, err := e.run([]string{"--manifest"}, "")
	if err != nil {
		return nil, fmt.Errorf("get manifest: %w", err)
	}
	return out, nil
}

func (e *Executor) GetExample() (string, error) {
	out, err := e.run([]string{"--example"}, "")
	if err != nil {
		return "", fmt.Errorf("get example: %w", err)
	}
	return string(out), nil
}
