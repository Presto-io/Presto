package main

import (
	"testing"

	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
)

// helper: find submenu by label among top-level items
func findSubmenu(m *menu.Menu, label string) *menu.Menu {
	for _, item := range m.Items {
		if item.Label == label && item.SubMenu != nil {
			return item.SubMenu
		}
	}
	return nil
}

// helper: count non-separator text items in a submenu
func textItems(m *menu.Menu) []*menu.MenuItem {
	var items []*menu.MenuItem
	for _, item := range m.Items {
		if item.Type == menu.TextType {
			items = append(items, item)
		}
	}
	return items
}

// helper: find text item by label
func findItem(m *menu.Menu, label string) *menu.MenuItem {
	for _, item := range m.Items {
		if item.Label == label {
			return item
		}
	}
	return nil
}

// helper: check accelerator equality
func accelEqual(a, b *keys.Accelerator) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if a.Key != b.Key {
		return false
	}
	if len(a.Modifiers) != len(b.Modifiers) {
		return false
	}
	for i := range a.Modifiers {
		if a.Modifiers[i] != b.Modifiers[i] {
			return false
		}
	}
	return true
}

func TestBuildMenu_TopLevelCount(t *testing.T) {
	m := buildMenu(&App{})

	// Expect 5 top-level items: AppMenu + 文件 + 编辑 + 模板 + 帮助
	if got := len(m.Items); got != 5 {
		t.Errorf("expected 5 top-level items, got %d", got)
		for i, item := range m.Items {
			t.Logf("  [%d] Label=%q Role=%v Type=%v", i, item.Label, item.Role, item.Type)
		}
	}
}

func TestBuildMenu_NoViewMenu(t *testing.T) {
	m := buildMenu(&App{})

	if sub := findSubmenu(m, "视图"); sub != nil {
		t.Error("menu should not contain a '视图' (View) submenu")
	}
}

func TestBuildMenu_NoWindowMenu(t *testing.T) {
	m := buildMenu(&App{})

	if sub := findSubmenu(m, "窗口"); sub != nil {
		t.Error("menu should not contain a standalone '窗口' (Window) submenu")
	}
}

func TestFileMenu_Items(t *testing.T) {
	m := buildMenu(&App{})
	fileMenu := findSubmenu(m, "文件")
	if fileMenu == nil {
		t.Fatal("文件 menu not found")
	}

	// Expected items in order (including separators):
	// 新建, 打开文件…, 保存, 另存为…, 导出 PDF…, 设置…, sep, 最小化, 缩放, sep, 退出
	expectedLabels := []string{
		"新建", "打开文件…", "保存", "另存为…", "导出 PDF…", "设置…",
		"", // separator
		"最小化", "缩放",
		"", // separator
		"退出",
	}

	if got := len(fileMenu.Items); got != len(expectedLabels) {
		t.Fatalf("文件 menu: expected %d items, got %d", len(expectedLabels), got)
	}

	for i, expected := range expectedLabels {
		item := fileMenu.Items[i]
		if expected == "" {
			// separator
			if item.Type != menu.SeparatorType {
				t.Errorf("文件 menu[%d]: expected separator, got %q (type=%v)", i, item.Label, item.Type)
			}
		} else {
			if item.Label != expected {
				t.Errorf("文件 menu[%d]: expected label %q, got %q", i, expected, item.Label)
			}
		}
	}
}

func TestFileMenu_Accelerators(t *testing.T) {
	m := buildMenu(&App{})
	fileMenu := findSubmenu(m, "文件")
	if fileMenu == nil {
		t.Fatal("文件 menu not found")
	}

	tests := []struct {
		label string
		accel *keys.Accelerator
	}{
		{"新建", keys.CmdOrCtrl("n")},
		{"打开文件…", keys.CmdOrCtrl("o")},
		{"保存", keys.CmdOrCtrl("s")},
		{"另存为…", keys.Combo("s", keys.CmdOrCtrlKey, keys.ShiftKey)},
		{"导出 PDF…", keys.CmdOrCtrl("e")},
		{"设置…", keys.CmdOrCtrl(",")},
		{"最小化", keys.CmdOrCtrl("m")},
		{"退出", keys.CmdOrCtrl("w")},
	}

	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			item := findItem(fileMenu, tt.label)
			if item == nil {
				t.Fatalf("item %q not found in 文件 menu", tt.label)
			}
			if !accelEqual(item.Accelerator, tt.accel) {
				t.Errorf("item %q: accelerator mismatch\n  got:  %+v\n  want: %+v",
					tt.label, item.Accelerator, tt.accel)
			}
		})
	}
}

func TestEditMenu(t *testing.T) {
	m := buildMenu(&App{})

	// EditMenu is a role-based item (Role=EditMenuRole), not a labeled submenu.
	// It should be the 3rd item (index 2): AppMenu, 文件, 编辑, 模板, 帮助
	found := false
	for _, item := range m.Items {
		if item.Role == menu.EditMenuRole {
			found = true
			break
		}
	}
	if !found {
		t.Error("编辑 menu (EditMenuRole) not found in top-level items")
	}
}

func TestTemplateMenu_Items(t *testing.T) {
	m := buildMenu(&App{})
	tmplMenu := findSubmenu(m, "模板")
	if tmplMenu == nil {
		t.Fatal("模板 menu not found")
	}

	items := textItems(tmplMenu)
	if got := len(items); got != 2 {
		t.Fatalf("模板 menu: expected 2 text items, got %d", got)
	}

	// Item 1: 模板商店 (no accelerator)
	if items[0].Label != "模板商店" {
		t.Errorf("模板 menu[0]: expected '模板商店', got %q", items[0].Label)
	}
	if items[0].Accelerator != nil {
		t.Errorf("模板商店 should have no accelerator, got %+v", items[0].Accelerator)
	}

	// Item 2: 模板管理… (CmdOrCtrl+Shift+T)
	if items[1].Label != "模板管理…" {
		t.Errorf("模板 menu[1]: expected '模板管理…', got %q", items[1].Label)
	}
	expectedAccel := keys.Combo("t", keys.CmdOrCtrlKey, keys.ShiftKey)
	if !accelEqual(items[1].Accelerator, expectedAccel) {
		t.Errorf("模板管理… accelerator mismatch\n  got:  %+v\n  want: %+v",
			items[1].Accelerator, expectedAccel)
	}
}

func TestHelpMenu_Items(t *testing.T) {
	m := buildMenu(&App{})
	helpMenu := findSubmenu(m, "帮助")
	if helpMenu == nil {
		t.Fatal("帮助 menu not found")
	}

	items := textItems(helpMenu)
	if got := len(items); got != 3 {
		t.Fatalf("帮助 menu: expected 3 text items, got %d", got)
	}

	expectedLabels := []string{"文档", "关于 Presto", "检查更新"}
	for i, expected := range expectedLabels {
		if items[i].Label != expected {
			t.Errorf("帮助 menu[%d]: expected %q, got %q", i, expected, items[i].Label)
		}
		// All help menu items have no accelerator
		if items[i].Accelerator != nil {
			t.Errorf("帮助 menu item %q should have no accelerator", items[i].Label)
		}
	}
}
