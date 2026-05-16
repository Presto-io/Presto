package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func (a *App) CompileSVG(typstSource string, workDir string) ([]string, error) {
	return a.compiler.CompileToSVG(typstSource, workDir)
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

			devDist := filepath.Join(exeDir, "..", "dist", name)
			if isRegularFile(devDist) {
				return devDist
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
