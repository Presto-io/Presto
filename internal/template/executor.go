package template

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

type Executor struct {
	BinaryPath string
}

func NewExecutor(binaryPath string) *Executor {
	return &Executor{BinaryPath: binaryPath}
}

func (e *Executor) Convert(markdown string) (string, error) {
	cmd := exec.Command(e.BinaryPath)
	cmd.Stdin = strings.NewReader(markdown)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("template execution failed: %w\nstderr: %s", err, stderr.String())
	}
	return stdout.String(), nil
}

func (e *Executor) GetManifest() ([]byte, error) {
	cmd := exec.Command(e.BinaryPath, "--manifest")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("manifest retrieval failed: %w\nstderr: %s", err, stderr.String())
	}
	return stdout.Bytes(), nil
}

func (e *Executor) GetExample() (string, error) {
	cmd := exec.Command(e.BinaryPath, "--example")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("example retrieval failed: %w\nstderr: %s", err, stderr.String())
	}
	return stdout.String(), nil
}
