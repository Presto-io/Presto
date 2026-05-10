package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestFindTypstBinaryFromWindowsPrefersBundledExe(t *testing.T) {
	exeDir := t.TempDir()
	typstPath := filepath.Join(exeDir, "typst.exe")
	if err := os.WriteFile(typstPath, []byte("stub"), 0755); err != nil {
		t.Fatalf("write typst.exe: %v", err)
	}

	got := findTypstBinaryFrom(exeDir, "windows", func(name string) (string, error) {
		t.Fatalf("lookPath should not be called when bundled typst.exe exists, got %q", name)
		return "", errors.New("unreachable")
	})

	if got != typstPath {
		t.Fatalf("expected bundled typst.exe path %q, got %q", typstPath, got)
	}
}

func TestFindTypstBinaryFromWindowsUsesPathExe(t *testing.T) {
	want := filepath.Join("C:", "Tools", "typst.exe")

	got := findTypstBinaryFrom("", "windows", func(name string) (string, error) {
		if name == "typst.exe" {
			return want, nil
		}
		return "", exec.ErrNotFound
	})

	if got != want {
		t.Fatalf("expected PATH typst.exe %q, got %q", want, got)
	}
}

func TestFindTypstBinaryFromWindowsDoesNotBypassErrDot(t *testing.T) {
	got := findTypstBinaryFrom("", "windows", func(name string) (string, error) {
		if name == "typst.exe" {
			return `.\typst.exe`, exec.ErrDot
		}
		return "", exec.ErrNotFound
	})

	if got != "typst" {
		t.Fatalf("expected fallback to naked typst, got %q", got)
	}
}

func TestTypstBinaryCandidates(t *testing.T) {
	windows := typstBinaryCandidates("windows")
	if len(windows) != 2 || windows[0] != "typst.exe" || windows[1] != "typst" {
		t.Fatalf("unexpected windows candidates: %#v", windows)
	}

	other := typstBinaryCandidates("darwin")
	if len(other) != 1 || other[0] != "typst" {
		t.Fatalf("unexpected darwin candidates: %#v", other)
	}
}

func TestExportPDFBaseNameUsesJiaoanFrontMatter(t *testing.T) {
	markdown := `---
template: "jiaoan-shicao"
course_name: "电工基本技能训练"
course_attribute: "基本技能课程"
textbook_name: "电工基本技能训练指导"
class_name: "机电技术应用 1 班"
total_hours: "8"
teacher_name: "张老师"
use_time: "2026 年 5 月 12 日 —— 2026 年 5 月 15 日"
---

## 学习任务分析
`

	got := exportPDFBaseName(markdown, "jiaoan-shicao", "")
	want := "教学设计方案 电工基本技能训练 8H"
	if got != want {
		t.Fatalf("exportPDFBaseName() = %q, want %q", got, want)
	}
}

func TestExportPDFBaseNameKeepsExistingHourSuffix(t *testing.T) {
	markdown := `---
course_name: PLC 综合实训
total_hours: 8H
---
`

	got := exportPDFBaseName(markdown, "jiaoan-shicao", "")
	want := "教学设计方案 PLC 综合实训 8H"
	if got != want {
		t.Fatalf("exportPDFBaseName() = %q, want %q", got, want)
	}
}

func TestExportPDFBaseNameFallsBackToTypstTitle(t *testing.T) {
	markdown := `---
course_name: 不应影响其他模板
total_hours: 8
---
`

	got := exportPDFBaseName(markdown, "gongwen", "= 公文标题\n")
	if got != "公文标题" {
		t.Fatalf("exportPDFBaseName() = %q, want %q", got, "公文标题")
	}
}
