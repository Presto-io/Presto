package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/mrered/presto/internal/preview"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const previewFallbackMessage = "兼容预览"

type runningPreviewProcess struct {
	cmd          *exec.Cmd
	cancel       context.CancelFunc
	workDir      string
	dataPlaneURL string
	controlConn  *websocket.Conn
	controlURL   string
}

type previewRunner struct {
	service         *preview.Service
	tinymistPath    string
	maxRetries      int
	retryBase       time.Duration
	cmd             *exec.Cmd
	sessionWorkDir  string
	dataPlaneURL    string
	controlPlaneURL string
	controlConn     *websocket.Conn
	cancel          context.CancelFunc
	retryCount      int
	mu              sync.Mutex
	controlMu       sync.Mutex
	updateMu        sync.Mutex
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
		"--partial-rendering=false",
		"--input=presto_fast_preview=true",
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
		r.mu.Lock()
		if r.cmd == cmd {
			r.cmd = nil
			r.cancel = nil
		}
		r.mu.Unlock()
	}()
	return nil
}

func (r *previewRunner) stop() error {
	return stopRunningPreviewProcess(r.detachProcess())
}

func (r *previewRunner) detachProcess() runningPreviewProcess {
	r.mu.Lock()
	process := runningPreviewProcess{
		cmd:          r.cmd,
		cancel:       r.cancel,
		workDir:      r.sessionWorkDir,
		dataPlaneURL: r.dataPlaneURL,
		controlConn:  r.controlConn,
		controlURL:   r.controlPlaneURL,
	}
	r.cmd = nil
	r.cancel = nil
	r.sessionWorkDir = ""
	r.dataPlaneURL = ""
	r.controlConn = nil
	r.controlPlaneURL = ""
	r.mu.Unlock()

	return process
}

func (r *previewRunner) restoreProcess(process runningPreviewProcess) {
	if process.cmd == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.cmd != nil {
		_ = stopRunningPreviewProcess(process)
		return
	}
	r.cmd = process.cmd
	r.cancel = process.cancel
	r.sessionWorkDir = process.workDir
	r.dataPlaneURL = process.dataPlaneURL
	r.controlPlaneURL = process.controlURL
	r.controlConn = process.controlConn
}

func stopRunningPreviewProcess(process runningPreviewProcess) error {
	if process.cancel != nil {
		process.cancel()
	}
	if process.cmd != nil && process.cmd.Process != nil {
		_ = process.cmd.Process.Kill()
	}
	if process.controlConn != nil {
		_ = process.controlConn.Close()
	}
	if process.workDir != "" {
		return os.RemoveAll(process.workDir)
	}
	return nil
}

func (r *previewRunner) hasProcess() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.cmd != nil
}

func (r *previewRunner) setDataPlaneURL(dataPlaneURL string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.dataPlaneURL = dataPlaneURL
}

func (r *previewRunner) setControlPlane(controlPlaneURL string, conn *websocket.Conn) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.controlPlaneURL = controlPlaneURL
	r.controlConn = conn
}

func (r *previewRunner) controlConnSnapshot() *websocket.Conn {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.controlConn
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

	if !a.previewService.IsCurrentVersion(result.Version) {
		return &result, nil
	}

	a.previewRunner.updateMu.Lock()
	defer a.previewRunner.updateMu.Unlock()

	if !a.previewService.IsCurrentVersion(result.Version) {
		return &result, nil
	}

	var previousProcess runningPreviewProcess
	if result.RestartSession {
		if logger != nil {
			logger.Info("[preview] restarting tinymist session", "version", result.Version, "reason", "document identity changed")
		}
		if workDir == "" {
			previousProcess = a.previewRunner.detachProcess()
		} else {
			_ = a.previewRunner.stop()
		}
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
		if previousProcess.cmd != nil {
			_ = a.previewRunner.stop()
			a.previewRunner.restoreProcess(previousProcess)
			previousProcess = runningPreviewProcess{}
		}
		return a.applyFallback(&result, typstSource, workDir, reason)
	}

	if a.tinymistUnavailable() {
		return applyFallback("tinymist binary not found")
	}

	if result.RestartSession || !a.previewRunner.hasProcess() {
		sessionMainTypPath := identity.MainTypstPath
		startEvent, ok := a.previewService.StartSessionForVersion(result.Version, preview.DocumentIdentity{
			TemplateID:    templateID,
			WorkDir:       workDir,
			DocumentKey:   documentKey,
			MainTypstPath: sessionMainTypPath,
		})
		if !ok {
			return &result, nil
		}
		result.Events = append(result.Events, startEvent)
		a.emitPreviewEvent(startEvent)

		dataPort, controlPort, err := allocatePreviewPorts()
		if err != nil {
			return applyFallback(err.Error())
		}

		if err := a.previewRunner.startTinymist(context.Background(), mainTypPath, dataPort, controlPort); err != nil {
			return applyFallback(err.Error())
		}
		dataPlaneURL := fmt.Sprintf("http://127.0.0.1:%d", dataPort)
		controlPlaneURL := fmt.Sprintf("ws://127.0.0.1:%d", controlPort)
		if err := waitForPreviewDataPlane(context.Background(), dataPlaneURL, 5*time.Second, 100*time.Millisecond); err != nil {
			_ = a.previewRunner.stop()
			a.previewRunner.restoreProcess(previousProcess)
			previousProcess = runningPreviewProcess{}
			if !a.previewService.IsCurrentVersion(result.Version) {
				return &result, nil
			}
			return applyFallback(fmt.Sprintf("tinymist preview not ready: %v", err))
		}
		controlConn, err := connectTinymistControlPlane(context.Background(), controlPlaneURL, 5*time.Second, 100*time.Millisecond)
		if err != nil {
			_ = a.previewRunner.stop()
			a.previewRunner.restoreProcess(previousProcess)
			previousProcess = runningPreviewProcess{}
			if !a.previewService.IsCurrentVersion(result.Version) {
				return &result, nil
			}
			return applyFallback(fmt.Sprintf("tinymist control plane not ready: %v", err))
		}
		if !a.previewService.IsCurrentVersion(result.Version) {
			_ = controlConn.Close()
			_ = a.previewRunner.stop()
			a.previewRunner.restoreProcess(previousProcess)
			return &result, nil
		}

		readyEvent, ok := a.previewService.ApplySessionEventForVersion(
			result.Version,
			a.previewService.CurrentSessionID(),
			preview.EventReady,
			dataPlaneURL,
			nil,
		)
		if !ok {
			_ = a.previewRunner.stop()
			a.previewRunner.restoreProcess(previousProcess)
			return &result, nil
		}
		result.Events = append(result.Events, readyEvent)
		a.emitPreviewEvent(readyEvent)
		a.previewRunner.setDataPlaneURL(dataPlaneURL)
		a.previewRunner.setControlPlane(controlPlaneURL, controlConn)
		go drainTinymistControlPlane(controlConn)

		if previousProcess.cmd != nil {
			_ = stopRunningPreviewProcess(previousProcess)
			previousProcess = runningPreviewProcess{}
		}
	} else {
		if err := a.previewRunner.updateTinymistMemoryFile(context.Background(), mainTypPath, typstSource, 2*time.Second); err != nil {
			if logger != nil {
				logger.Warn("[preview] tinymist refresh failed", "version", result.Version, "error", err)
			}
			return applyFallback(fmt.Sprintf("tinymist preview refresh failed: %v", err))
		}
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
	unavailable, ok := a.previewService.TinymistUnavailableForVersion(result.Version, reason)
	if !ok {
		return result, nil
	}
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

func waitForPreviewDataPlane(ctx context.Context, url string, timeout time.Duration, pollInterval time.Duration) error {
	if pollInterval <= 0 {
		pollInterval = 100 * time.Millisecond
	}
	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	client := &http.Client{Timeout: pollInterval}
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	var lastErr error
	for {
		req, err := http.NewRequestWithContext(waitCtx, http.MethodGet, url, nil)
		if err != nil {
			return err
		}
		resp, err := client.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 500 {
				return nil
			}
			lastErr = fmt.Errorf("unexpected status %d", resp.StatusCode)
		} else {
			lastErr = err
		}

		select {
		case <-waitCtx.Done():
			if lastErr != nil {
				return lastErr
			}
			return waitCtx.Err()
		case <-ticker.C:
		}
	}
}

func connectTinymistControlPlane(ctx context.Context, controlPlaneURL string, timeout time.Duration, pollInterval time.Duration) (*websocket.Conn, error) {
	if controlPlaneURL == "" {
		return nil, fmt.Errorf("missing tinymist control plane URL")
	}
	if pollInterval <= 0 {
		pollInterval = 100 * time.Millisecond
	}

	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	var lastErr error
	for {
		conn, _, err := websocket.DefaultDialer.DialContext(waitCtx, controlPlaneURL, nil)
		if err == nil {
			return conn, nil
		}
		lastErr = err

		select {
		case <-waitCtx.Done():
			if lastErr != nil {
				return nil, lastErr
			}
			return nil, waitCtx.Err()
		case <-ticker.C:
		}
	}
}

func drainTinymistControlPlane(conn *websocket.Conn) {
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			return
		}
		var response struct {
			Event string `json:"event"`
			Kind  string `json:"kind"`
		}
		if err := json.Unmarshal(message, &response); err == nil && logger != nil {
			switch response.Event {
			case "compileStatus":
				logger.Debug("[preview] tinymist compile status", "status", response.Kind)
			case "syncEditorChanges":
				logger.Debug("[preview] tinymist requested editor memory sync")
			case "outline", "editorScrollTo":
			default:
				logger.Debug("[preview] tinymist control message", "event", response.Event)
			}
		}
	}
}

func (r *previewRunner) updateTinymistMemoryFile(ctx context.Context, mainTypPath string, typstSource string, timeout time.Duration) error {
	conn := r.controlConnSnapshot()
	if conn == nil {
		return fmt.Errorf("missing tinymist control plane connection")
	}

	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	message := map[string]interface{}{
		"event": "updateMemoryFiles",
		"files": map[string]string{
			mainTypPath: typstSource,
		},
	}

	r.controlMu.Lock()
	defer r.controlMu.Unlock()

	if deadline, ok := waitCtx.Deadline(); ok {
		_ = conn.SetWriteDeadline(deadline)
	}
	return conn.WriteJSON(message)
}
