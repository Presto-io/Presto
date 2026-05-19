package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
)

func (a *App) CompileSVG(typstSource string, workDir string) ([]string, error) {
	return a.compiler.CompileToSVG(typstSource, workDir)
}

func findTypstBinary() string {
	return findTypstBinaryWithDataDir("")
}

func findTypstBinaryWithDataDir(dataDir string) string {
	exeDir := ""
	exe, err := os.Executable()
	if err == nil {
		exe, _ = filepath.EvalSymlinks(exe)
		exeDir = filepath.Dir(exe)
	}

	return findTypstBinaryFrom(exeDir, dataDir, runtime.GOOS, exec.LookPath)
}

func findTypstBinaryFrom(exeDir string, dataDir string, goos string, lookPath func(string) (string, error)) string {
	candidates := typstBinaryCandidates(goos)

	if packaged := findPackagedTypstBinary(exeDir, goos); packaged != "" {
		return packaged
	}

	if runtimeBin := findUserDataRuntimeBinary(dataDir, "typst", candidates, goos+"-"+runtime.GOARCH); runtimeBin != "" {
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
		}
	}

	return "typst"
}

func findPackagedTypstBinary(exeDir string, goos string) string {
	if exeDir == "" {
		return ""
	}
	for _, name := range typstBinaryCandidates(goos) {
		if goos == "darwin" {
			resources := filepath.Join(exeDir, "..", "Resources", name)
			if isRegularFile(resources) {
				return resources
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

func findUserDataRuntimeBinary(dataDir string, tool string, candidates []string, platform string) string {
	if dataDir == "" {
		return ""
	}
	root := filepath.Join(dataDir, "runtimes", tool)
	entries, err := os.ReadDir(root)
	if err != nil {
		return ""
	}
	var versions []string
	for _, entry := range entries {
		if entry.IsDir() {
			versions = append(versions, entry.Name())
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(versions)))
	for _, version := range versions {
		for _, name := range candidates {
			candidate := filepath.Join(root, version, platform, name)
			if isRegularFile(candidate) {
				return candidate
			}
		}
	}
	return ""
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
