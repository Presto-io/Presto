package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func findTinymistBinary() string {
	return findTinymistBinaryWithDataDir("")
}

func findTinymistBinaryWithDataDir(dataDir string) string {
	exeDir := ""
	exe, err := os.Executable()
	if err == nil {
		exe, _ = filepath.EvalSymlinks(exe)
		exeDir = filepath.Dir(exe)
	}

	return findTinymistBinaryFrom(exeDir, dataDir, runtime.GOOS, runtime.GOARCH, exec.LookPath)
}

func findTinymistBinaryFrom(exeDir string, dataDir string, goos string, goarch string, lookPath func(string) (string, error)) string {
	candidates := tinymistBinaryCandidates(goos)

	if packaged := findPackagedTinymistBinary(exeDir, goos, goarch); packaged != "" {
		return packaged
	}

	// User data runtimes live under DataDir/runtimes/tinymist.
	if runtimeBin := findUserDataRuntimeBinary(dataDir, "tinymist", candidates, goos+"-"+goarch); runtimeBin != "" {
		return runtimeBin
	}

	if exeDir != "" {
		for _, name := range candidates {
			devDist := filepath.Join(exeDir, "..", "dist", name)
			if isRegularFile(devDist) {
				return devDist
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

func findPackagedTinymistBinary(exeDir string, goos string, goarch string) string {
	if exeDir == "" {
		return ""
	}
	for _, name := range tinymistBinaryCandidates(goos) {
		if goos == "darwin" {
			resources := filepath.Join(exeDir, "..", "Resources", name)
			if isRegularFile(resources) {
				return resources
			}
			sidecar := filepath.Join(exeDir, "..", "Resources", "sidecars", "tinymist", goos+"-"+goarch, name)
			if isRegularFile(sidecar) {
				return sidecar
			}
			continue
		}
		beside := filepath.Join(exeDir, name)
		if isRegularFile(beside) {
			return beside
		}
	}
	return ""
}

func tinymistBinaryCandidates(goos string) []string {
	if goos == "windows" {
		return []string{"tinymist.exe", "tinymist"}
	}
	return []string{"tinymist"}
}
