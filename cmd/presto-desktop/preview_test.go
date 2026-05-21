package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/mrered/presto/internal/preview"
	"github.com/mrered/presto/internal/template"
	"github.com/mrered/presto/internal/typst"
)

func TestPreviewRunnerWritesSessionMainTyp(t *testing.T) {
	runner := newPreviewRunner(preview.NewService(), "tinymist")
	mainTypPath, cleanup, err := runner.writeSessionFile("", "#show: doc")
	if err != nil {
		t.Fatalf("write session file: %v", err)
	}
	defer cleanup()

	if filepath.Base(mainTypPath) != "main.typ" {
		t.Fatalf("main typ path = %q, want main.typ", mainTypPath)
	}

	data, err := os.ReadFile(mainTypPath)
	if err != nil {
		t.Fatalf("read main.typ: %v", err)
	}
	if string(data) != "#show: doc" {
		t.Fatalf("main.typ = %q", string(data))
	}
	if runner.sessionWorkDir == "" {
		t.Fatal("runner should track owned session workdir")
	}
}

func TestPreviewRunnerWritesWorkDirSessionMainTypInSystemTemp(t *testing.T) {
	runner := newPreviewRunner(preview.NewService(), "tinymist")
	documentDir := t.TempDir()
	mainTypPath, cleanup, err := runner.writeSessionFile(documentDir, "#show: doc")
	if err != nil {
		t.Fatalf("write session file: %v", err)
	}
	defer cleanup()

	if filepath.Dir(mainTypPath) == documentDir {
		t.Fatalf("main.typ should be written outside document dir, got %q", mainTypPath)
	}
	if _, err := os.Stat(filepath.Join(documentDir, "main.typ")); !os.IsNotExist(err) {
		t.Fatalf("document dir main.typ should not exist, stat err = %v", err)
	}
	if filepath.Dir(mainTypPath) != runner.sessionWorkDir {
		t.Fatalf("runner sessionWorkDir = %q, want %q", runner.sessionWorkDir, filepath.Dir(mainTypPath))
	}
}

func TestPreviewRunnerBuildsTinymistArgs(t *testing.T) {
	runner := newPreviewRunner(preview.NewService(), "tinymist")
	got := runner.buildTinymistArgs("/tmp/main.typ", 23625, 23626)
	want := []string{
		"preview",
		"/tmp/main.typ",
		"--no-open",
		"--partial-rendering=false",
		"--input=presto_fast_preview=true",
		"--data-plane-host=127.0.0.1:23625",
		"--control-plane-host=127.0.0.1:23626",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("args = %#v, want %#v", got, want)
	}
}

func TestPreviewRunnerRetryDelayCapsAtFive(t *testing.T) {
	runner := newPreviewRunner(preview.NewService(), "tinymist")
	want := []time.Duration{
		500 * time.Millisecond,
		time.Second,
		2 * time.Second,
		4 * time.Second,
		8 * time.Second,
	}

	for i, expected := range want {
		got, ok := runner.nextRetryDelay()
		if !ok {
			t.Fatalf("retry %d unexpectedly capped", i+1)
		}
		if got != expected {
			t.Fatalf("retry %d delay = %s, want %s", i+1, got, expected)
		}
	}

	if got, ok := runner.nextRetryDelay(); ok || got != 0 {
		t.Fatalf("retry 6 = (%s, %v), want capped", got, ok)
	}
}

func TestPreviewRunnerStopIsIdempotent(t *testing.T) {
	runner := newPreviewRunner(preview.NewService(), "tinymist")
	workDir := t.TempDir()
	ownedDir := filepath.Join(workDir, "owned")
	if err := os.Mkdir(ownedDir, 0755); err != nil {
		t.Fatalf("create owned dir: %v", err)
	}
	runner.sessionWorkDir = ownedDir

	if err := runner.stop(); err != nil {
		t.Fatalf("first stop: %v", err)
	}
	if _, err := os.Stat(ownedDir); !os.IsNotExist(err) {
		t.Fatalf("owned dir should be removed, stat err = %v", err)
	}
	if err := runner.stop(); err != nil {
		t.Fatalf("second stop: %v", err)
	}
}

func TestPreviewStopForcesNextSameIdentityUpdateToRestart(t *testing.T) {
	service := preview.NewService()
	runner := newPreviewRunner(service, "tinymist")
	app := NewApp(nil, nil, nil, ReleaseCapabilities{}, service, runner)

	identity := preview.DocumentIdentity{TemplateID: "mock", DocumentKey: "doc.md"}
	first := service.BeginUpdate(identity)
	if event, ok := service.StartSessionForVersion(first.Version, identity); !ok {
		t.Fatalf("start session failed: %#v", event)
	}
	if event, ok := service.ApplySessionEventForVersion(first.Version, service.CurrentSessionID(), preview.EventReady, "http://127.0.0.1:1", nil); !ok {
		t.Fatalf("ready session failed: %#v", event)
	}

	if err := app.PreviewStop(); err != nil {
		t.Fatalf("PreviewStop returned error: %v", err)
	}

	second := service.BeginUpdate(identity)
	if !second.RestartSession {
		t.Fatal("same document update after PreviewStop should restart the resident preview session")
	}
}

func TestPreviewRunnerStopsOldSessionBeforeNewDocumentIdentity(t *testing.T) {
	runner := newPreviewRunner(preview.NewService(), "tinymist")
	oldMainTypPath, oldCleanup, err := runner.writeSessionFile("", "#let doc = old")
	if err != nil {
		t.Fatalf("write old session: %v", err)
	}
	defer oldCleanup()
	oldWorkDir := runner.sessionWorkDir
	if oldWorkDir == "" {
		t.Fatal("old session workdir should be tracked")
	}

	if err := runner.stop(); err != nil {
		t.Fatalf("stop old session: %v", err)
	}
	if _, err := os.Stat(oldMainTypPath); !os.IsNotExist(err) {
		t.Fatalf("old session main.typ should be removed before new session, stat err = %v", err)
	}

	newMainTypPath, newCleanup, err := runner.writeSessionFile("", "#let doc = new")
	if err != nil {
		t.Fatalf("write new session: %v", err)
	}
	defer newCleanup()
	if runner.sessionWorkDir == "" {
		t.Fatal("new session workdir should be tracked")
	}
	if runner.sessionWorkDir == oldWorkDir {
		t.Fatalf("new session reused old workdir %q after document identity switch", oldWorkDir)
	}
	if newMainTypPath == oldMainTypPath {
		t.Fatalf("new session retained old main.typ path %q", newMainTypPath)
	}
	if _, err := os.Stat(newMainTypPath); err != nil {
		t.Fatalf("new session main.typ should exist: %v", err)
	}
}

func TestPreviewRunnerRewritesExistingSessionFileForContentUpdate(t *testing.T) {
	runner := newPreviewRunner(preview.NewService(), "tinymist")
	mainTypPath, cleanup, err := runner.writeSessionFile("", "#let doc = old")
	if err != nil {
		t.Fatalf("write initial session: %v", err)
	}
	defer cleanup()
	workDir := runner.sessionWorkDir

	nextMainTypPath, _, err := runner.writeSessionFile("", "#let doc = new")
	if err != nil {
		t.Fatalf("rewrite session: %v", err)
	}
	if nextMainTypPath != mainTypPath {
		t.Fatalf("content update should reuse main.typ path: got %q want %q", nextMainTypPath, mainTypPath)
	}
	if runner.sessionWorkDir != workDir {
		t.Fatalf("content update should reuse workdir: got %q want %q", runner.sessionWorkDir, workDir)
	}
	data, err := os.ReadFile(mainTypPath)
	if err != nil {
		t.Fatalf("read rewritten main.typ: %v", err)
	}
	if string(data) != "#let doc = new" {
		t.Fatalf("main.typ = %q, want new content", string(data))
	}
}

func TestPreviewRunnerStartTinymistReturnsStartError(t *testing.T) {
	runner := newPreviewRunner(preview.NewService(), filepath.Join(t.TempDir(), "missing-tinymist"))
	if err := runner.startTinymist(context.Background(), "/tmp/main.typ", "", 1, 2); err == nil {
		t.Fatal("startTinymist should return start error for missing binary")
	}
}

func TestPreviewRunnerStopsBrokenSessionAfterRefreshFailure(t *testing.T) {
	templateRoot := t.TempDir()
	writeDesktopPreviewTestTemplate(t, templateRoot, "mock")

	compiler := typst.NewCompiler()
	compiler.BinPath = filepath.Join(t.TempDir(), "missing-typst")
	service := preview.NewService()
	runner := newPreviewRunner(service, "tinymist")
	app := NewApp(
		template.NewManager(templateRoot),
		compiler,
		nil,
		ReleaseCapabilities{},
		service,
		runner,
	)

	sessionDir := t.TempDir()
	runner.sessionWorkDir = sessionDir
	runner.cmd = exec.Command("go", "version")
	closedControlURL, closedControlConn := startClosedControlPlane(t)
	runner.setControlPlane(closedControlURL, closedControlConn)

	first := service.BeginUpdate(preview.DocumentIdentity{TemplateID: "mock", DocumentKey: "doc.md"})
	if event, ok := service.StartSessionForVersion(first.Version, preview.DocumentIdentity{TemplateID: "mock", DocumentKey: "doc.md"}); !ok {
		t.Fatalf("start session failed: %#v", event)
	}
	if event, ok := service.ApplySessionEventForVersion(first.Version, service.CurrentSessionID(), preview.EventReady, "http://127.0.0.1:1", nil); !ok {
		t.Fatalf("ready session failed: %#v", event)
	}

	second := service.BeginUpdate(preview.DocumentIdentity{TemplateID: "mock", DocumentKey: "doc.md"})
	if _, err := app.finishPreviewUpdate(&second, "#let doc = new", "mock", "", "doc.md"); err == nil {
		t.Fatal("finishPreviewUpdate should return fallback compile error after tinymist refresh failure")
	}
	if runner.hasProcess() {
		t.Fatal("broken tinymist session should be stopped so next preview can restart it")
	}
	if runner.sessionWorkDir != "" {
		t.Fatalf("sessionWorkDir = %q, want cleared", runner.sessionWorkDir)
	}
	if _, err := os.Stat(sessionDir); !os.IsNotExist(err) {
		t.Fatalf("broken session dir should be removed, stat err = %v", err)
	}
}

func TestWaitForPreviewDataPlaneWaitsUntilReady(t *testing.T) {
	var ready atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !ready.Load() {
			http.Error(w, "starting", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	go func() {
		time.Sleep(30 * time.Millisecond)
		ready.Store(true)
	}()

	if err := waitForPreviewDataPlane(context.Background(), server.URL, time.Second, 10*time.Millisecond); err != nil {
		t.Fatalf("waitForPreviewDataPlane returned error: %v", err)
	}
}

func startClosedControlPlane(t *testing.T) (string, *websocket.Conn) {
	t.Helper()
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	accepted := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade websocket: %v", err)
			return
		}
		close(accepted)
		_ = conn.Close()
	}))
	t.Cleanup(server.Close)

	controlURL := "ws" + server.URL[len("http"):]
	conn, _, err := websocket.DefaultDialer.Dial(controlURL, nil)
	if err != nil {
		t.Fatalf("dial closed control plane: %v", err)
	}
	<-accepted
	_ = conn.Close()
	return controlURL, conn
}

func TestWaitForPreviewDataPlaneTimesOut(t *testing.T) {
	port, err := allocateLocalPort()
	if err != nil {
		t.Fatalf("allocate port: %v", err)
	}

	err = waitForPreviewDataPlane(context.Background(), "http://127.0.0.1:"+portString(port), 50*time.Millisecond, 10*time.Millisecond)
	if err == nil {
		t.Fatal("waitForPreviewDataPlane should time out when data plane never listens")
	}
}

func TestPreviewRunnerUpdateTinymistMemoryFileSendsControlPlaneUpdate(t *testing.T) {
	gotMessage := make(chan map[string]interface{}, 1)
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade websocket: %v", err)
			return
		}
		defer conn.Close()

		var message map[string]interface{}
		if err := conn.ReadJSON(&message); err != nil {
			t.Errorf("read websocket json: %v", err)
			return
		}
		gotMessage <- message
	}))
	defer server.Close()

	controlURL := "ws" + server.URL[len("http"):]
	conn, _, err := websocket.DefaultDialer.Dial(controlURL, nil)
	if err != nil {
		t.Fatalf("dial control websocket: %v", err)
	}
	defer conn.Close()

	runner := newPreviewRunner(preview.NewService(), "tinymist")
	runner.setControlPlane(controlURL, conn)

	if err := runner.updateTinymistMemoryFile(context.Background(), "/tmp/main.typ", "#let doc = new", time.Second); err != nil {
		t.Fatalf("updateTinymistMemoryFile returned error: %v", err)
	}

	select {
	case got := <-gotMessage:
		if got["event"] != "updateMemoryFiles" {
			t.Fatalf("event = %v, want updateMemoryFiles", got["event"])
		}
		files, ok := got["files"].(map[string]interface{})
		if !ok {
			t.Fatalf("files = %#v, want object", got["files"])
		}
		if files["/tmp/main.typ"] != "#let doc = new" {
			t.Fatalf("main.typ = %v, want updated source", files["/tmp/main.typ"])
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for control plane update")
	}
}

func TestPreviewUpdateAsyncReturnsBeforeFallbackCompile(t *testing.T) {
	templateRoot := t.TempDir()
	writeDesktopPreviewTestTemplate(t, templateRoot, "mock")

	compiler := typst.NewCompiler()
	compiler.BinPath = filepath.Join(t.TempDir(), "missing-typst")
	app := NewApp(
		template.NewManager(templateRoot),
		compiler,
		nil,
		ReleaseCapabilities{},
		preview.NewService(),
		newPreviewRunner(preview.NewService(), filepath.Join(t.TempDir(), "missing-tinymist")),
	)

	start := time.Now()
	result, err := app.PreviewUpdateAsync("# Hello", "mock", "", "doc.md")
	if err != nil {
		t.Fatalf("PreviewUpdateAsync returned error: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 200*time.Millisecond {
		t.Fatalf("PreviewUpdateAsync took %s, expected immediate return", elapsed)
	}
	if result.Version == 0 {
		t.Fatal("expected document version to be assigned")
	}
	if len(result.Events) == 0 || result.Events[0].Mode != preview.ModeFallback {
		t.Fatalf("initial events = %#v, want fallback/status event", result.Events)
	}
}

func writeDesktopPreviewTestTemplate(t *testing.T, root string, name string) {
	t.Helper()
	tplDir := filepath.Join(root, name)
	if err := os.MkdirAll(tplDir, 0755); err != nil {
		t.Fatal(err)
	}
	manifest := `{"name":"` + name + `","displayName":"Mock","version":"0.1.0","author":"test"}`
	if err := os.WriteFile(filepath.Join(tplDir, "manifest.json"), []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}

	src := filepath.Join(t.TempDir(), "template.go")
	binName := "presto-template-" + name
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}
	bin := filepath.Join(tplDir, binName)
	code := `package main
import (
	"io"
	"os"
)
func main() {
	data, _ := io.ReadAll(os.Stdin)
	os.Stdout.Write(data)
}
`
	if err := os.WriteFile(src, []byte(code), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "build", "-o", bin, src)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build mock template: %v\n%s", err, output)
	}
}
