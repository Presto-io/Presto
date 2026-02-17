package typst

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestCompile(t *testing.T) {
	if _, err := exec.LookPath("typst"); err != nil {
		t.Skip("typst not installed, skipping")
	}

	dir := t.TempDir()
	typFile := filepath.Join(dir, "test.typ")
	os.WriteFile(typFile, []byte("= Hello World"), 0644)

	c := NewCompiler()
	pdfPath, err := c.Compile(typFile)
	if err != nil {
		t.Fatalf("compile failed: %v", err)
	}

	if _, err := os.Stat(pdfPath); err != nil {
		t.Fatalf("PDF not created: %v", err)
	}
}

func TestCompileFromString(t *testing.T) {
	if _, err := exec.LookPath("typst"); err != nil {
		t.Skip("typst not installed, skipping")
	}

	c := NewCompiler()
	pdf, err := c.CompileString("= Hello World", "")
	if err != nil {
		t.Fatalf("compile failed: %v", err)
	}
	if len(pdf) == 0 {
		t.Fatal("empty PDF output")
	}
	if string(pdf[:5]) != "%PDF-" {
		t.Fatalf("not a valid PDF, got header: %q", string(pdf[:5]))
	}
}
