package typst

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Compiler struct {
	Root    string // root directory for typst path resolution
	BinPath string // path to typst binary (empty = use PATH)
}

func NewCompiler() *Compiler {
	return &Compiler{}
}

func NewCompilerWithRoot(root string) *Compiler {
	return &Compiler{Root: root}
}

func (c *Compiler) typstBin() string {
	if c.BinPath != "" {
		return c.BinPath
	}
	return "typst"
}

func (c *Compiler) Compile(typFile string) (string, error) {
	pdfFile := strings.TrimSuffix(typFile, ".typ") + ".pdf"
	args := []string{"compile"}
	if c.Root != "" {
		args = append(args, "--root", c.Root)
	}
	args = append(args, typFile, pdfFile)
	cmd := exec.Command(c.typstBin(), args...)

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
