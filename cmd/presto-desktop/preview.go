package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/mrered/presto/internal/preview"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const previewFallbackMessage = "兼容预览"

type previewRunner struct {
	service         *preview.Service
	tinymistPath    string
	maxRetries      int
	retryBase       time.Duration
	cmd             *exec.Cmd
	sessionWorkDir  string
	cancel          context.CancelFunc
	retryCount      int
	mu              sync.Mutex
	currentMode     preview.Mode
	documentVersion int64
}

func newPreviewRunner(service *preview.Service, tinymistPath string) *previewRunner {
	return &previewRunner{service: service, tinymistPath: tinymistPath, maxRetries: 5, retryBase: 500 * time.Millisecond, currentMode: preview.ModeFallback}
}

func (r *previewRunner) writeSessionFile(workDir string, typstSource string) (mainTypPath string, cleanup func(), err error) {
	ownedWorkDir := false
	if workDir == "" {
		r.mu.Lock()
		workDir = r.sessionWorkDir
		r.mu.Unlock()
		if workDir == "" {
			workDir, err = os.MkdirTemp("", "presto-preview-*")
			if err != nil {
				return "", nil, err
			}
			ownedWorkDir = true
		} else {
			ownedWorkDir = true
		}
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

func (r *previewRunner) hasProcess() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.cmd != nil
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

func mainTypstIdentityPath(workDir string) string {
	if workDir == "" {
		return ""
	}
	return filepath.Join(workDir, "main.typ")
}

func (r *previewRunner) markEvent(event preview.Event) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if event.Mode != "" {
		r.currentMode = event.Mode
	}
	if event.DocumentVersion != 0 {
		r.documentVersion = event.DocumentVersion
	}
}

func (r *previewRunner) modeSnapshot() map[string]interface{} {
	r.mu.Lock()
	defer r.mu.Unlock()

	mode := r.currentMode
	if mode == "" {
		mode = preview.ModeFallback
	}
	sessionID := ""
	if r.service != nil {
		sessionID = r.service.CurrentSessionID()
	}
	return map[string]interface{}{
		"mode":            mode,
		"sessionId":       sessionID,
		"documentVersion": r.documentVersion,
		"retryCount":      r.retryCount,
		"tinymistPath":    r.tinymistPath,
		"fallbackMessage": previewFallbackMessage,
	}
}

func (a *App) PreviewUpdate(markdown string, templateID string, workDir string, documentKey string) (*preview.UpdateResult, error) {
	identity := preview.DocumentIdentity{
		TemplateID:    templateID,
		WorkDir:       workDir,
		DocumentKey:   documentKey,
		MainTypstPath: mainTypstIdentityPath(workDir),
	}
	result := a.previewService.BeginUpdate(identity)
	a.emitPreviewEvents(result.Events)

	tpl, err := a.manager.Get(templateID)
	if err != nil {
		event := a.previewService.ConversionFailed(result.Version, err)
		result.Events = append(result.Events, event)
		a.emitPreviewEvent(event)
		return &result, err
	}

	typstSource, err := a.manager.Executor(tpl).Convert(markdown)
	if err != nil {
		event := a.previewService.ConversionFailed(result.Version, err)
		result.Events = append(result.Events, event)
		a.emitPreviewEvent(event)
		return &result, err
	}

	if result.RestartSession {
		_ = a.previewRunner.stop()
	}

	mainTypPath, cleanup, err := a.previewRunner.writeSessionFile(workDir, typstSource)
	if err != nil {
		event := a.previewService.FallbackFailed(result.Version, err)
		result.Events = append(result.Events, event)
		a.emitPreviewEvent(event)
		return &result, err
	}
	applyFallback := func(reason string) (*preview.UpdateResult, error) {
		if cleanup != nil {
			cleanup()
		}
		return a.applyFallback(&result, typstSource, workDir, reason)
	}

	if a.tinymistUnavailable() {
		return applyFallback("tinymist binary not found")
	}

	if result.RestartSession || !a.previewRunner.hasProcess() {
		sessionMainTypPath := identity.MainTypstPath
		startEvent := a.previewService.StartSession(preview.DocumentIdentity{
			TemplateID:    templateID,
			WorkDir:       workDir,
			DocumentKey:   documentKey,
			MainTypstPath: sessionMainTypPath,
		})
		result.Events = append(result.Events, startEvent)
		a.emitPreviewEvent(startEvent)

		dataPort, controlPort, err := allocatePreviewPorts()
		if err != nil {
			return applyFallback(err.Error())
		}

		if err := a.previewRunner.startTinymist(context.Background(), mainTypPath, dataPort, controlPort); err != nil {
			return applyFallback(err.Error())
		}

		readyEvent, _ := a.previewService.ApplySessionEvent(
			a.previewService.CurrentSessionID(),
			preview.EventReady,
			fmt.Sprintf("http://127.0.0.1:%d", dataPort),
			nil,
		)
		result.Events = append(result.Events, readyEvent)
		a.emitPreviewEvent(readyEvent)
	}

	return &result, nil
}

func (a *App) PreviewStop() error {
	if a.previewRunner == nil {
		return nil
	}
	err := a.previewRunner.stop()
	if a.previewService != nil {
		event, _ := a.previewService.ApplySessionEvent(a.previewService.CurrentSessionID(), preview.EventTeardown, "", nil)
		a.emitPreviewEvent(event)
	}
	return err
}

func (a *App) PreviewMode() map[string]interface{} {
	if a.previewRunner == nil {
		return map[string]interface{}{
			"mode":            preview.ModeFallback,
			"sessionId":       "",
			"documentVersion": int64(0),
			"retryCount":      0,
			"tinymistPath":    "",
			"fallbackMessage": previewFallbackMessage,
		}
	}
	return a.previewRunner.modeSnapshot()
}

func (a *App) tinymistUnavailable() bool {
	if a.previewRunner == nil {
		return true
	}
	if a.previewRunner.tinymistPath != "tinymist" {
		return false
	}
	_, err := exec.LookPath("tinymist")
	return err != nil
}

func (a *App) applyFallback(result *preview.UpdateResult, typstSource string, workDir string, reason string) (*preview.UpdateResult, error) {
	unavailable := a.previewService.TinymistUnavailable(reason)
	result.Events = append(result.Events, unavailable)
	a.emitPreviewEvent(unavailable)

	pages, err := preview.CompileFallback(context.Background(), a.compiler, typstSource, workDir)
	if err != nil {
		event := a.previewService.FallbackFailed(result.Version, err)
		result.Events = append(result.Events, event)
		a.emitPreviewEvent(event)
		return result, err
	}

	event, _ := a.previewService.ApplyFallback(result.Version, pages)
	result.Events = append(result.Events, event)
	a.emitPreviewEvent(event)
	return result, nil
}

func (a *App) emitPreviewEvents(events []preview.Event) {
	for _, event := range events {
		a.emitPreviewEvent(event)
	}
}

func (a *App) emitPreviewEvent(event preview.Event) {
	if a.previewRunner != nil {
		a.previewRunner.markEvent(event)
	}
	if a.ctx != nil {
		wailsRuntime.EventsEmit(a.ctx, "preview:event", event)
	}
}

func allocatePreviewPorts() (int, int, error) {
	dataPort, err := allocateLocalPort()
	if err != nil {
		return 0, 0, err
	}
	controlPort, err := allocateLocalPort()
	if err != nil {
		return 0, 0, err
	}
	if dataPort == controlPort {
		return allocatePreviewPorts()
	}
	return dataPort, controlPort, nil
}

func allocateLocalPort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()
	addr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		return 0, fmt.Errorf("unexpected preview listener address: %s", listener.Addr())
	}
	return addr.Port, nil
}
