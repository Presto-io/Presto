package template

import "strings"

var filenameReplacer = strings.NewReplacer(
	"/", "_",
	"\\", "_",
	":", "_",
	"*", "_",
	"?", "_",
	`"`, "_",
	"<", "_",
	">", "_",
	"|", "_",
)

func IsGenericOutputBaseName(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "output", "untitled", "未命名":
		return true
	default:
		return false
	}
}

func CleanFilenameBase(value string, fallback string) string {
	value = strings.Join(strings.Fields(filenameReplacer.Replace(value)), " ")
	value = strings.Trim(value, ". _\t\r\n")
	if IsGenericOutputBaseName(value) {
		return fallback
	}
	return value
}

func OutputBaseNameOrMarkdownFallback(info OutputInfo, markdown string) string {
	candidates := []string{
		info.OutputBaseName,
		info.PreviewTitle,
		info.Document.Title,
		markdownTitle(markdown),
	}
	for _, candidate := range candidates {
		cleaned := CleanFilenameBase(candidate, "")
		if !IsGenericOutputBaseName(cleaned) {
			return cleaned
		}
	}
	return "presto-document"
}

func markdownTitle(markdown string) string {
	if title := frontMatterField(markdown, "title"); title != "" {
		return title
	}
	for _, line := range strings.Split(markdown, "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "# "))
		}
	}
	return ""
}

func frontMatterField(markdown string, field string) string {
	markdown = strings.TrimLeft(markdown, "\ufeff \t\r\n")
	if !strings.HasPrefix(markdown, "---") {
		return ""
	}
	lines := strings.Split(markdown, "\n")
	if len(lines) < 3 || strings.TrimSpace(lines[0]) != "---" {
		return ""
	}
	prefix := field + ":"
	for i := 1; i < len(lines); i++ {
		line := strings.TrimSpace(strings.TrimSuffix(lines[i], "\r"))
		if line == "---" || line == "..." {
			break
		}
		if !strings.HasPrefix(line, prefix) {
			continue
		}
		value := strings.TrimSpace(strings.TrimPrefix(line, prefix))
		if strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`) && len(value) >= 2 {
			value = strings.Trim(value, `"`)
		} else if strings.HasPrefix(value, `'`) && strings.HasSuffix(value, `'`) && len(value) >= 2 {
			value = strings.Trim(value, `'`)
		}
		if idx := strings.Index(value, " #"); idx > 0 {
			value = strings.TrimSpace(value[:idx])
		}
		return value
	}
	return ""
}
