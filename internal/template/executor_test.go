package template

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func createMockTemplate(t *testing.T, dir string) string {
	t.Helper()
	src := filepath.Join(dir, "mock.go")
	bin := filepath.Join(dir, "mock-template")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}

	code := `package main
import (
	"fmt"
	"io"
	"os"
)
func main() {
	if len(os.Args) > 1 && os.Args[1] == "--manifest" {
		fmt.Print(` + "`" + `{"name":"mock","version":"0.1.0"}` + "`" + `)
		return
	}
	data, _ := io.ReadAll(os.Stdin)
	fmt.Printf("// converted\n%s", data)
}
`
	os.WriteFile(src, []byte(code), 0644)
	cmd := exec.Command("go", "build", "-o", bin, src)
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to build mock template: %v", err)
	}
	return bin
}

func TestExecute(t *testing.T) {
	dir := t.TempDir()
	bin := createMockTemplate(t, dir)

	ex := NewExecutor(bin)
	result, err := ex.Convert("# Hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "// converted\n# Hello" {
		t.Errorf("got %q, want %q", result, "// converted\n# Hello")
	}
}

func TestExecuteManifest(t *testing.T) {
	dir := t.TempDir()
	bin := createMockTemplate(t, dir)

	ex := NewExecutor(bin)
	data, err := ex.GetManifest()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m, err := ParseManifest(data)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if m.Name != "mock" {
		t.Errorf("got name %q, want %q", m.Name, "mock")
	}
}
