package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

var letPattern = regexp.MustCompile(`#let\s+(\w+)\s*=\s*"([^"]*)"`)

func exportPDFBaseName(markdown string, templateID string, typstOutput string) string {
	if name := jiaoanShicaoPDFBaseName(markdown, templateID); name != "" {
		return name
	}
	return extractTypstTitle(typstOutput)
}

func jiaoanShicaoPDFBaseName(markdown string, templateID string) string {
	if templateID != "jiaoan-shicao" {
		return ""
	}
	fields := extractSimpleFrontMatter(markdown)
	courseName := strings.TrimSpace(fields["course_name"])
	totalHours := normalizeHourLabel(fields["total_hours"])
	if courseName == "" || totalHours == "" {
		return ""
	}
	return sanitizeFilename("教学设计方案 " + courseName + " " + totalHours)
}

func extractSimpleFrontMatter(markdown string) map[string]string {
	result := map[string]string{}
	trimmed := strings.TrimLeft(markdown, "\ufeff \t\r\n")
	if !strings.HasPrefix(trimmed, "---") {
		return result
	}
	rest := trimmed[3:]
	if strings.HasPrefix(rest, "\r\n") {
		rest = rest[2:]
	} else if strings.HasPrefix(rest, "\n") {
		rest = rest[1:]
	}

	end := strings.Index(rest, "\n---")
	if end < 0 {
		return result
	}
	for _, line := range strings.Split(rest[:end], "\n") {
		key, value, ok := strings.Cut(strings.TrimSpace(line), ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(stripInlineYAMLComment(value))
		if len(value) >= 2 {
			first, last := value[0], value[len(value)-1]
			if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
				value = value[1 : len(value)-1]
			}
		}
		result[key] = value
	}
	return result
}

func stripInlineYAMLComment(value string) string {
	inSingleQuote := false
	inDoubleQuote := false
	escaped := false
	for i, r := range value {
		if escaped {
			escaped = false
			continue
		}
		if r == '\\' && inDoubleQuote {
			escaped = true
			continue
		}
		if r == '\'' && !inDoubleQuote {
			inSingleQuote = !inSingleQuote
			continue
		}
		if r == '"' && !inSingleQuote {
			inDoubleQuote = !inDoubleQuote
			continue
		}
		if r == '#' && !inSingleQuote && !inDoubleQuote {
			if i == 0 || value[i-1] == ' ' || value[i-1] == '\t' {
				return strings.TrimSpace(value[:i])
			}
		}
	}
	return value
}

func normalizeHourLabel(totalHours string) string {
	hours := strings.TrimSpace(totalHours)
	if hours == "" {
		return ""
	}
	upper := strings.ToUpper(hours)
	if strings.HasSuffix(upper, "H") || strings.Contains(hours, "课时") || strings.Contains(hours, "小时") {
		return hours
	}
	return hours + "H"
}

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
	exeDir := ""
	exe, err := os.Executable()
	if err == nil {
		exe, _ = filepath.EvalSymlinks(exe)
		exeDir = filepath.Dir(exe)
	}

	return findTypstBinaryFrom(exeDir, runtime.GOOS, exec.LookPath)
}

func findTypstBinaryFrom(exeDir string, goos string, lookPath func(string) (string, error)) string {
	candidates := typstBinaryCandidates(goos)

	if exeDir != "" {
		for _, name := range candidates {
			resources := filepath.Join(exeDir, "..", "Resources", name)
			if isRegularFile(resources) {
				return resources
			}

			beside := filepath.Join(exeDir, name)
			if isRegularFile(beside) {
				return beside
			}
		}
	}

	for _, name := range candidates {
		if p, err := lookPath(name); err == nil {
			return p
		}
	}

	return "typst"
}

func typstBinaryCandidates(goos string) []string {
	if goos == "windows" {
		return []string{"typst.exe", "typst"}
	}
	return []string{"typst"}
}

func isRegularFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
