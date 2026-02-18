package typst

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
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

// CompileString compiles typst source to PDF.
// If workDir is non-empty, the temp .typ file is written there so relative
// paths (e.g. images) resolve from the document's directory.
func (c *Compiler) CompileString(typstSource, workDir string) ([]byte, error) {
	if workDir != "" {
		typFile := filepath.Join(workDir, ".presto-temp-input.typ")
		if err := os.WriteFile(typFile, []byte(typstSource), 0644); err != nil {
			return nil, err
		}
		defer os.Remove(typFile)

		pdfFile, err := c.Compile(typFile)
		if err != nil {
			return nil, err
		}
		defer os.Remove(pdfFile)
		return os.ReadFile(pdfFile)
	}

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
// If workDir is non-empty, relative paths resolve from that directory.
func (c *Compiler) CompileToSVG(typstSource, workDir string) ([]string, error) {
	var dir string
	var cleanDir bool

	if workDir != "" {
		dir = workDir
	} else {
		var err error
		dir, err = os.MkdirTemp("", "presto-svg-*")
		if err != nil {
			return nil, err
		}
		cleanDir = true
	}
	if cleanDir {
		defer os.RemoveAll(dir)
	}

	typFile := filepath.Join(dir, ".presto-temp-input.typ")
	if err := os.WriteFile(typFile, []byte(typstSource), 0644); err != nil {
		return nil, err
	}
	if !cleanDir {
		defer os.Remove(typFile)
	}

	// typst compile --format svg outputs {name}-{page}.svg for multi-page
	outPattern := filepath.Join(dir, ".presto-temp-output-{p}.svg")
	args := []string{"compile", "--format", "svg"}
	if c.Root != "" {
		args = append(args, "--root", c.Root)
	}
	args = append(args, typFile, outPattern)
	cmd := exec.Command(c.typstBin(), args...)
	log.Printf("[compile-svg] running: %s %s", c.typstBin(), strings.Join(args, " "))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("typst svg compile failed: %w\noutput: %s", err, output)
	}
	if len(output) > 0 {
		log.Printf("[compile-svg] typst output: %s", output)
	}

	// Collect SVG pages — use glob as primary method for robustness
	globPattern := filepath.Join(dir, ".presto-temp-output-*.svg")
	matches, _ := filepath.Glob(globPattern)
	sort.Strings(matches) // lexicographic sort works for single-digit; re-sort numerically below

	// Sort numerically by extracting page number
	sort.Slice(matches, func(i, j int) bool {
		ni, nj := 0, 0
		fmt.Sscanf(filepath.Base(matches[i]), ".presto-temp-output-%d.svg", &ni)
		fmt.Sscanf(filepath.Base(matches[j]), ".presto-temp-output-%d.svg", &nj)
		return ni < nj
	})

	var pages []string
	for _, svgFile := range matches {
		data, err := os.ReadFile(svgFile)
		if err != nil {
			continue
		}
		pages = append(pages, string(data))
		if !cleanDir {
			os.Remove(svgFile)
		}
	}
	if len(pages) == 0 {
		// Single page fallback: try without page number
		svgFile := filepath.Join(dir, ".presto-temp-output.svg")
		data, err := os.ReadFile(svgFile)
		if err == nil {
			pages = append(pages, string(data))
			if !cleanDir {
				os.Remove(svgFile)
			}
		}
	}
	if len(pages) == 0 {
		// List directory for diagnostics
		entries, _ := os.ReadDir(dir)
		var names []string
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), ".svg") {
				names = append(names, e.Name())
			}
		}
		return nil, fmt.Errorf("no SVG output produced (dir=%s, svg_files=%v, typst_output=%s)", dir, names, output)
	}
	return pages, nil
}
