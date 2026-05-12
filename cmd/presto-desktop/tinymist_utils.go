package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func findTinymistBinary() string {
	exeDir := ""
	exe, err := os.Executable()
	if err == nil {
		exe, _ = filepath.EvalSymlinks(exe)
		exeDir = filepath.Dir(exe)
	}

	return findTinymistBinaryFrom(exeDir, runtime.GOOS, runtime.GOARCH, exec.LookPath)
}

func findTinymistBinaryFrom(exeDir string, goos string, goarch string, lookPath func(string) (string, error)) string {
	candidates := tinymistBinaryCandidates(goos)

	if exeDir != "" {
		for _, name := range candidates {
			sidecar := filepath.Join(exeDir, "..", "Resources", "sidecars", "tinymist", goos+"-"+goarch, name)
			if isRegularFile(sidecar) {
				return sidecar
			}
		}

		for _, name := range candidates {
			resources := filepath.Join(exeDir, "..", "Resources", name)
			if isRegularFile(resources) {
				return resources
			}
		}

		for _, name := range candidates {
			beside := filepath.Join(exeDir, name)
			if isRegularFile(beside) {
				return beside
			}
		}
	}

	for _, name := range candidates {
		if p, err := lookPath(name); err == nil {
			return p
		} else if errors.Is(err, exec.ErrDot) {
			continue
		}
	}

	return "tinymist"
}

func tinymistBinaryCandidates(goos string) []string {
	if goos == "windows" {
		return []string{"tinymist.exe", "tinymist"}
	}
	return []string{"tinymist"}
}
