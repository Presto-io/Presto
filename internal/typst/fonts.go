package typst

import (
	"regexp"
	"strings"
)

var (
	fontLetPattern    = regexp.MustCompile(`(#let\s+FONT_[A-Za-z0-9_]+\s*=\s*)(\([^)\n]*\)|"[^"\n]*")`)
	fontOptionPattern = regexp.MustCompile(`(font:\s*)(\([^)\n]*\)|"[^"\n]*")`)
	quotedFontPattern = regexp.MustCompile(`"([^"\n]*)"`)
)

func HasCompatibleFont(name string, available map[string]bool) bool {
	_, ok := resolveAvailableFont(name, available)
	return ok
}

func normalizeTypstFontFamilies(source string, available map[string]bool) string {
	if len(available) == 0 {
		return source
	}
	source = fontLetPattern.ReplaceAllStringFunc(source, func(match string) string {
		parts := fontLetPattern.FindStringSubmatch(match)
		if len(parts) != 3 {
			return match
		}
		if literal, ok := chooseAvailableFontLiteral(parts[2], available); ok {
			return parts[1] + literal
		}
		return match
	})
	source = fontOptionPattern.ReplaceAllStringFunc(source, func(match string) string {
		parts := fontOptionPattern.FindStringSubmatch(match)
		if len(parts) != 3 {
			return match
		}
		if literal, ok := chooseAvailableFontLiteral(parts[2], available); ok {
			return parts[1] + literal
		}
		return match
	})
	return source
}

func chooseAvailableFontLiteral(literal string, available map[string]bool) (string, bool) {
	matches := quotedFontPattern.FindAllStringSubmatch(literal, -1)
	for _, match := range matches {
		if len(match) != 2 {
			continue
		}
		if name, ok := resolveAvailableFont(match[1], available); ok {
			return `"` + name + `"`, true
		}
	}
	return "", false
}

func resolveAvailableFont(name string, available map[string]bool) (string, bool) {
	if exact, ok := canonicalFontName(name, available); ok {
		return exact, true
	}
	for _, alias := range fontAliases(name) {
		if exact, ok := canonicalFontName(alias, available); ok {
			return exact, true
		}
	}
	return "", false
}

func canonicalFontName(name string, available map[string]bool) (string, bool) {
	if available[name] {
		return name, true
	}
	folded := strings.ToLower(name)
	for candidate := range available {
		if strings.ToLower(candidate) == folded {
			return candidate, true
		}
	}
	return "", false
}

func fontAliases(name string) []string {
	switch strings.ToLower(name) {
	case "songti sc", "stsong", "simsun", "nsimsun":
		return []string{"STSong", "Songti SC", "SimSun", "NSimSun", "Noto Serif CJK SC"}
	case "stfangsong", "fangsong":
		return []string{"STFangsong", "FangSong"}
	case "stheiti", "simhei":
		return []string{"STHeiti", "SimHei", "Microsoft YaHei"}
	case "stkaiti", "kaiti":
		return []string{"STKaiti", "KaiTi"}
	case "fzxiaobiaosong-b05":
		return []string{"FZXiaoBiaoSong-B05", "STZhongsong", "STSong", "SimSun"}
	case "noto serif cjk sc":
		return []string{"Noto Serif CJK SC", "STSong", "Songti SC", "SimSun"}
	case "noto sans cjk sc":
		return []string{"Noto Sans CJK SC", "Microsoft YaHei", "SimHei", "STHeiti"}
	default:
		return nil
	}
}
