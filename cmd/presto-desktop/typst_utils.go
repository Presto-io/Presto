package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var letPattern = regexp.MustCompile(`#let\s+(\w+)\s*=\s*"([^"]*)"`)

func (a *App) CompileSVG(typstSource string, workDir string) ([]string, error) {
	return a.compiler.CompileToSVG(typstSource, workDir)
}

func extractTypstTitle(typ string) string {
	lines := strings.Split(typ, "\n")
	for level := 1; level <= 5; level++ {
		prefix := strings.Repeat("=", level) + " "
		deeperPrefix := strings.Repeat("=", level+1)
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if !strings.HasPrefix(trimmed, prefix) {
				continue
			}
			if level < 5 && strings.HasPrefix(trimmed, deeperPrefix) {
				continue
			}
			content := strings.TrimSpace(trimmed[len(prefix):])
			title := resolveTypstText(content, lines)
			title = sanitizeFilename(title)
			if title != "" {
				return title
			}
		}
	}
	return "output"
}

func resolveTypstText(content string, lines []string) string {
	if !strings.HasPrefix(content, "#") {
		return content
	}
	varName := content[1:]
	if idx := strings.IndexAny(varName, ".( "); idx > 0 {
		varName = varName[:idx]
	}
	for _, line := range lines {
		m := letPattern.FindStringSubmatch(line)
		if m != nil && m[1] == varName {
			return m[2]
		}
	}
	return ""
}

func sanitizeFilename(s string) string {
	return strings.Map(func(r rune) rune {
		if strings.ContainsRune(`/\:*?"<>|`, r) {
			return '_'
		}
		return r
	}, strings.TrimSpace(s))
}

func findTypstBinary() string {
	exe, err := os.Executable()
	if err == nil {
		exe, _ = filepath.EvalSymlinks(exe)
		exeDir := filepath.Dir(exe)

		resources := filepath.Join(exeDir, "..", "Resources", "typst")
		if _, err := os.Stat(resources); err == nil {
			return resources
		}

		beside := filepath.Join(exeDir, "typst")
		if _, err := os.Stat(beside); err == nil {
			return beside
		}
	}

	if p, err := exec.LookPath("typst"); err == nil {
		return p
	}

	return "typst"
}
