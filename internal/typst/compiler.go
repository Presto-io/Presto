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

// CompileToSVG compiles typst source to SVG pages.
// Returns a slice of SVG strings, one per page.
func (c *Compiler) CompileToSVG(typstSource string) ([]string, error) {
	dir, err := os.MkdirTemp("", "presto-svg-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)

	typFile := filepath.Join(dir, "input.typ")
	if err := os.WriteFile(typFile, []byte(typstSource), 0644); err != nil {
		return nil, err
	}

	// typst compile --format svg outputs {name}-{page}.svg for multi-page
	outPattern := filepath.Join(dir, "output-{n}.svg")
	args := []string{"compile", "--format", "svg"}
	if c.Root != "" {
		args = append(args, "--root", c.Root)
	}
	args = append(args, typFile, outPattern)
	cmd := exec.Command(c.typstBin(), args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("typst svg compile failed: %w\noutput: %s", err, output)
	}

	// Collect SVG pages
	var pages []string
	for i := 1; ; i++ {
		svgFile := filepath.Join(dir, fmt.Sprintf("output-%d.svg", i))
		data, err := os.ReadFile(svgFile)
		if err != nil {
			break
		}
		pages = append(pages, string(data))
	}
	if len(pages) == 0 {
		// Single page: try output-1.svg or output.svg
		data, err := os.ReadFile(filepath.Join(dir, "output.svg"))
		if err == nil {
			pages = append(pages, string(data))
		}
	}
	if len(pages) == 0 {
		return nil, fmt.Errorf("no SVG output produced")
	}
	return pages, nil
}
