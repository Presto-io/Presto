package typst

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const compileTimeout = 60 * time.Second // SEC-12

type Compiler struct {
	Root      string   // root directory for typst path resolution
	BinPath   string   // path to typst binary (empty = use PATH)
	FontPaths []string // additional font directories for --font-path
}

func NewCompiler() *Compiler {
	return &Compiler{}
}

func NewCompilerWithRoot(root string) *Compiler {
	// SEC-02: Warn if root is "/" — callers should use a restricted path
	if root == "/" {
		log.Printf("[typst] WARNING: compiler root set to \"/\" — this allows reading all files. Use a restricted directory.")
	}
	return &Compiler{Root: root}
}

func (c *Compiler) typstBin() string {
	if c.BinPath != "" {
		return c.BinPath
	}
	return "typst"
}

// ListFonts returns the set of available font family names.
func (c *Compiler) ListFonts() map[string]bool {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	args := []string{"fonts"}
	for _, fp := range c.FontPaths {
		args = append(args, "--font-path", fp)
	}
	cmd := exec.CommandContext(ctx, c.typstBin(), args...)
	output, err := cmd.Output()
	if err != nil {
		log.Printf("[typst] failed to list fonts: %v", err)
		return nil
	}
	fonts := make(map[string]bool)
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if name := strings.TrimSpace(line); name != "" {
			fonts[name] = true
		}
	}
	return fonts
}

// randomSuffix generates a cryptographically random hex string (SEC-25).
func randomSuffix() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func (c *Compiler) Compile(typFile string) (string, error) {
	return c.compileWithRoot(typFile, c.Root)
}

// compileWithRoot compiles a .typ file to PDF using the given root directory.
// The root must contain the source file, otherwise typst will reject it.
func (c *Compiler) compileWithRoot(typFile, root string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), compileTimeout)
	defer cancel()

	pdfFile := strings.TrimSuffix(typFile, ".typ") + ".pdf"
	args := []string{"compile"}
	if root != "" {
		args = append(args, "--root", root)
	}
	for _, fp := range c.FontPaths {
		args = append(args, "--font-path", fp)
	}
	args = append(args, typFile, pdfFile)
	cmd := exec.CommandContext(ctx, c.typstBin(), args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("typst compile timed out after %s", compileTimeout)
		}
		return "", fmt.Errorf("typst compile failed: %w\noutput: %s", err, output)
	}
	return pdfFile, nil
}

// CompileString compiles typst source to PDF.
// If workDir is non-empty, the temp .typ file is written there so relative
// paths (e.g. images) resolve from the document's directory.
func (c *Compiler) CompileString(typstSource, workDir string) ([]byte, error) {
	if workDir != "" {
		// SEC-25: Use random suffix to avoid race conditions
		typFile := filepath.Join(workDir, fmt.Sprintf(".presto-temp-%s.typ", randomSuffix()))
		if err := os.WriteFile(typFile, []byte(typstSource), 0644); err != nil {
			return nil, err
		}
		defer os.Remove(typFile)

		// Use workDir as root so the source file is inside the project root
		pdfFile, err := c.compileWithRoot(typFile, workDir)
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

	// Use the temp dir as root so the source file is inside the project root
	pdfFile, err := c.compileWithRoot(typFile, dir)
	if err != nil {
		return nil, err
	}

	return os.ReadFile(pdfFile)
}

// CompileToSVG compiles typst source to SVG pages.
// If workDir is non-empty, relative paths resolve from that directory.
func (c *Compiler) CompileToSVG(typstSource, workDir string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), compileTimeout)
	defer cancel()

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

	// SEC-25: Use random suffix to avoid race conditions
	suffix := randomSuffix()
	typFile := filepath.Join(dir, fmt.Sprintf(".presto-temp-%s.typ", suffix))
	if err := os.WriteFile(typFile, []byte(typstSource), 0644); err != nil {
		return nil, err
	}
	if !cleanDir {
		defer os.Remove(typFile)
	}

	// typst compile --format svg outputs {name}-{page}.svg for multi-page
	outPattern := filepath.Join(dir, fmt.Sprintf(".presto-temp-%s-{p}.svg", suffix))
	args := []string{"compile", "--format", "svg"}
	// Use the actual working directory as root so the source file
	// is always inside the project root (typst requires this).
	args = append(args, "--root", dir)
	for _, fp := range c.FontPaths {
		args = append(args, "--font-path", fp)
	}
	args = append(args, typFile, outPattern)
	cmd := exec.CommandContext(ctx, c.typstBin(), args...)
	log.Printf("[compile-svg] running: %s %s", c.typstBin(), strings.Join(args, " "))
	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("typst svg compile timed out after %s", compileTimeout)
		}
		return nil, fmt.Errorf("typst svg compile failed: %w\noutput: %s", err, output)
	}
	if len(output) > 0 {
		log.Printf("[compile-svg] typst output: %s", output)
	}

	// Collect SVG pages — use glob as primary method for robustness
	globPattern := filepath.Join(dir, fmt.Sprintf(".presto-temp-%s-*.svg", suffix))
	matches, _ := filepath.Glob(globPattern)
	sort.Strings(matches) // lexicographic sort works for single-digit; re-sort numerically below

	// Sort numerically by extracting page number
	scanFmt := fmt.Sprintf(".presto-temp-%s-%%d.svg", suffix)
	sort.Slice(matches, func(i, j int) bool {
		ni, nj := 0, 0
		fmt.Sscanf(filepath.Base(matches[i]), scanFmt, &ni)
		fmt.Sscanf(filepath.Base(matches[j]), scanFmt, &nj)
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
		svgFile := filepath.Join(dir, fmt.Sprintf(".presto-temp-%s.svg", suffix))
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
