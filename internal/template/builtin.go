package template

import "path/filepath"

func ResolveBuiltinTemplatesDir(exeDir string, goos string) string {
	if exeDir == "" {
		return ""
	}
	switch goos {
	case "darwin":
		return filepath.Join(exeDir, "..", "Resources", "templates")
	case "windows", "linux":
		return filepath.Join(exeDir, "templates")
	default:
		return filepath.Join(exeDir, "templates")
	}
}

func ResolveDevBuiltinTemplatesDir(exeDir string) string {
	if exeDir == "" {
		return ""
	}
	return filepath.Join(exeDir, "..", "..", "template-registry", "templates")
}
