package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/mrered/presto/internal/preview"
)

type previewRunner struct {
	service        *preview.Service
	tinymistPath   string
	maxRetries     int
	retryBase      time.Duration
	cmd            *exec.Cmd
	sessionWorkDir string
	cancel         context.CancelFunc
	retryCount     int
	mu             sync.Mutex
}

func newPreviewRunner(service *preview.Service, tinymistPath string) *previewRunner {
	return &previewRunner{service: service, tinymistPath: tinymistPath, maxRetries: 5, retryBase: 500 * time.Millisecond}
}

func (r *previewRunner) writeSessionFile(workDir string, typstSource string) (mainTypPath string, cleanup func(), err error) {
	ownedWorkDir := false
	if workDir == "" {
		workDir, err = os.MkdirTemp("", "presto-preview-*")
		if err != nil {
			return "", nil, err
		}
		ownedWorkDir = true
	}

	if err := os.MkdirAll(workDir, 0755); err != nil {
		if ownedWorkDir {
			_ = os.RemoveAll(workDir)
		}
		return "", nil, err
	}

	mainTypPath = filepath.Join(workDir, "main.typ")
	if err := os.WriteFile(mainTypPath, []byte(typstSource), 0644); err != nil {
		if ownedWorkDir {
			_ = os.RemoveAll(workDir)
		}
		return "", nil, err
	}

	if ownedWorkDir {
		r.mu.Lock()
		r.sessionWorkDir = workDir
		r.mu.Unlock()
	}

	cleanup = func() {
		if !ownedWorkDir {
			return
		}
		_ = os.RemoveAll(workDir)
		r.mu.Lock()
		if r.sessionWorkDir == workDir {
			r.sessionWorkDir = ""
		}
		r.mu.Unlock()
	}
	return mainTypPath, cleanup, nil
}

func (r *previewRunner) buildTinymistArgs(mainTypPath string, dataPort int, controlPort int) []string {
	return []string{
		"preview",
		mainTypPath,
		"--no-open",
		"--partial-rendering=true",
		"--data-plane-host=127.0.0.1:" + portString(dataPort),
		"--control-plane-host=127.0.0.1:" + portString(controlPort),
	}
}

func (r *previewRunner) startTinymist(ctx context.Context, mainTypPath string, dataPort int, controlPort int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	runCtx, cancel := context.WithCancel(ctx)
	cmd := exec.CommandContext(runCtx, r.tinymistPath, r.buildTinymistArgs(mainTypPath, dataPort, controlPort)...)
	if err := cmd.Start(); err != nil {
		cancel()
		return err
	}

	r.cancel = cancel
	r.cmd = cmd
	go func() {
		_ = cmd.Wait()
	}()
	return nil
}

func (r *previewRunner) stop() error {
	r.mu.Lock()
	cmd := r.cmd
	cancel := r.cancel
	workDir := r.sessionWorkDir
	r.cmd = nil
	r.cancel = nil
	r.sessionWorkDir = ""
	r.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
	if workDir != "" {
		return os.RemoveAll(workDir)
	}
	return nil
}

func (r *previewRunner) nextRetryDelay() (time.Duration, bool) {
	if r.retryCount >= r.maxRetries {
		return 0, false
	}
	delay := r.retryBase << r.retryCount
	r.retryCount++
	return delay, true
}

func portString(port int) string {
	return strconv.Itoa(port)
}
