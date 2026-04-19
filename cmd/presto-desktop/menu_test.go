package main

import (
	"testing"

	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
)

func findSubmenu(m *menu.Menu, label string) *menu.Menu {
	for _, item := range m.Items {
		if item.Label == label && item.SubMenu != nil {
			return item.SubMenu
		}
	}
	return nil
}

func textItems(m *menu.Menu) []*menu.MenuItem {
	var items []*menu.MenuItem
	for _, item := range m.Items {
		if item.Type == menu.TextType {
			items = append(items, item)
		}
	}
	return items
}

func findItem(m *menu.Menu, label string) *menu.MenuItem {
	for _, item := range m.Items {
		if item.Label == label {
			return item
		}
	}
	return nil
}

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

func hasTopLevelRole(m *menu.Menu, role menu.Role) bool {
	for _, item := range m.Items {
		if item.Role == role {
			return true
		}
	}
	return false
}

func requireTopLevelLabel(t *testing.T, item *menu.MenuItem, label string) {
	t.Helper()
	if item.Label != label {
		t.Fatalf("expected top-level label %q, got %q", label, item.Label)
	}
}

func assertSharedFileMenu(t *testing.T, m *menu.Menu) {
	t.Helper()
	fileMenu := findSubmenu(m, "文件")
	if fileMenu == nil {
		t.Fatal("文件 menu not found")
	}

	expectedLabels := []string{
		"新建", "打开文件…", "保存", "另存为…", "导出 PDF…", "设置…",
	}

	if got := len(fileMenu.Items); got != len(expectedLabels) {
		t.Fatalf("文件 menu: expected %d items, got %d", len(expectedLabels), got)
	}

	for i, expected := range expectedLabels {
		item := fileMenu.Items[i]
		if item.Label != expected {
			t.Errorf("文件 menu[%d]: expected label %q, got %q", i, expected, item.Label)
		}
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

func TestBuildMenu_DarwinStructure(t *testing.T) {
	m := buildMenuForPlatform(&App{}, "darwin")

	if got := len(m.Items); got != 6 {
		t.Fatalf("expected 6 top-level items for darwin, got %d", got)
	}

	if m.Items[0].Role != menu.AppMenuRole {
		t.Fatalf("expected first top-level item to be AppMenuRole, got role=%v", m.Items[0].Role)
	}
	requireTopLevelLabel(t, m.Items[1], "文件")
	if m.Items[2].Role != menu.EditMenuRole {
		t.Fatalf("expected third top-level item to be EditMenuRole, got role=%v", m.Items[2].Role)
	}
	requireTopLevelLabel(t, m.Items[3], "模板")
	if m.Items[4].Role != menu.WindowMenuRole {
		t.Fatalf("expected fifth top-level item to be WindowMenuRole, got role=%v", m.Items[4].Role)
	}
	requireTopLevelLabel(t, m.Items[5], "帮助")
	assertSharedFileMenu(t, m)
}

func TestBuildMenu_DarwinFileMenu_NoWindowItems(t *testing.T) {
	m := buildMenuForPlatform(&App{}, "darwin")
	fileMenu := findSubmenu(m, "文件")
	if fileMenu == nil {
		t.Fatal("文件 menu not found")
	}

	for _, label := range []string{"最小化", "缩放", "退出"} {
		if item := findItem(fileMenu, label); item != nil {
			t.Fatalf("darwin 文件 menu should not include %q", label)
		}
	}
}

func TestBuildMenu_DarwinHelpMenu_NoAbout(t *testing.T) {
	m := buildMenuForPlatform(&App{}, "darwin")
	helpMenu := findSubmenu(m, "帮助")
	if helpMenu == nil {
		t.Fatal("帮助 menu not found")
	}

	if item := findItem(helpMenu, "关于 Presto"); item != nil {
		t.Fatal("darwin 帮助 menu should not include 关于 Presto")
	}

	expectedLabels := []string{"文档", "检查更新"}
	items := textItems(helpMenu)
	if got := len(items); got != len(expectedLabels) {
		t.Fatalf("darwin 帮助 menu: expected %d items, got %d", len(expectedLabels), got)
	}
	for i, expected := range expectedLabels {
		if items[i].Label != expected {
			t.Errorf("darwin 帮助 menu[%d]: expected %q, got %q", i, expected, items[i].Label)
		}
	}
}

func TestBuildMenu_WindowsStructure(t *testing.T) {
	m := buildMenuForPlatform(&App{}, "windows")

	if got := len(m.Items); got != 4 {
		t.Fatalf("expected 4 top-level items for windows, got %d", got)
	}

	requireTopLevelLabel(t, m.Items[0], "文件")
	if m.Items[1].Role != menu.EditMenuRole {
		t.Fatalf("expected second top-level item to be EditMenuRole, got role=%v", m.Items[1].Role)
	}
	requireTopLevelLabel(t, m.Items[2], "模板")
	requireTopLevelLabel(t, m.Items[3], "帮助")

	if hasTopLevelRole(m, menu.AppMenuRole) {
		t.Fatal("windows menu should not include AppMenuRole")
	}
	if hasTopLevelRole(m, menu.WindowMenuRole) {
		t.Fatal("windows menu should not include WindowMenuRole")
	}

	assertSharedFileMenu(t, m)
}

func TestBuildMenu_WindowsHelpMenu_HasAbout(t *testing.T) {
	m := buildMenuForPlatform(&App{}, "windows")
	helpMenu := findSubmenu(m, "帮助")
	if helpMenu == nil {
		t.Fatal("帮助 menu not found")
	}

	items := textItems(helpMenu)
	expectedLabels := []string{"文档", "关于 Presto", "检查更新"}
	if got := len(items); got != len(expectedLabels) {
		t.Fatalf("windows 帮助 menu: expected %d items, got %d", len(expectedLabels), got)
	}
	for i, expected := range expectedLabels {
		if items[i].Label != expected {
			t.Errorf("windows 帮助 menu[%d]: expected %q, got %q", i, expected, items[i].Label)
		}
	}
}

func TestBuildMenu_WindowsFileMenu_NoWindowMenuItems(t *testing.T) {
	m := buildMenuForPlatform(&App{}, "windows")
	fileMenu := findSubmenu(m, "文件")
	if fileMenu == nil {
		t.Fatal("文件 menu not found")
	}

	for _, label := range []string{"最小化", "缩放", "退出"} {
		if item := findItem(fileMenu, label); item != nil {
			t.Fatalf("windows 文件 menu should not include %q", label)
		}
	}
}
