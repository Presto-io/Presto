package typst

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Compiler struct{}

func NewCompiler() *Compiler {
	return &Compiler{}
}

func (c *Compiler) Compile(typFile string) (string, error) {
	pdfFile := strings.TrimSuffix(typFile, ".typ") + ".pdf"
	cmd := exec.Command("typst", "compile", typFile, pdfFile)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("typst compile failed: %w\noutput: %s", err, output)
	}
	return pdfFile, nil
}

func (c *Compiler) CompileString(typstSource string) ([]byte, error) {
	dir, err := os.MkdirTemp("", "presto-compile-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)

	typFile := filepath.Join(dir, "input.typ")
	if err := os.WriteFile(typFile, []byte(typstSource), 0644); err != nil {
		return nil, err
	}

	pdfFile, err := c.Compile(typFile)
	if err != nil {
		return nil, err
	}

	return os.ReadFile(pdfFile)
}
