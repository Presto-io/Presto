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

	// 6 text items + 1 separator + 1 "关闭窗口" = 8 items total
	expectedLabels := []string{
		"新建", "打开文件…", "保存", "另存为…", "导出 PDF…", "设置…",
	}

	if got := len(fileMenu.Items); got != 8 {
		t.Fatalf("文件 menu: expected 8 items, got %d", got)
	}

	for i, expected := range expectedLabels {
		item := fileMenu.Items[i]
		if item.Label != expected {
			t.Errorf("文件 menu[%d]: expected label %q, got %q", i, expected, item.Label)
		}
	}

	// Item 6 is a separator
	if fileMenu.Items[6].Type != menu.SeparatorType {
		t.Errorf("文件 menu[6]: expected separator, got type=%v", fileMenu.Items[6].Type)
	}

	// Item 7 is "关闭窗口"
	if fileMenu.Items[7].Label != "关闭窗口" {
		t.Errorf("文件 menu[7]: expected label %q, got %q", "关闭窗口", fileMenu.Items[7].Label)
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
		{"关闭窗口", keys.CmdOrCtrl("w")},
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

	if got := len(m.Items); got != 7 {
		t.Fatalf("expected 7 top-level items for darwin, got %d", got)
	}

	if m.Items[0].Role != menu.AppMenuRole {
		t.Fatalf("expected first top-level item to be AppMenuRole, got role=%v", m.Items[0].Role)
	}
	requireTopLevelLabel(t, m.Items[1], "文件")
	if m.Items[2].Role != menu.EditMenuRole {
		t.Fatalf("expected third top-level item to be EditMenuRole, got role=%v", m.Items[2].Role)
	}
	requireTopLevelLabel(t, m.Items[3], "模板")
	requireTopLevelLabel(t, m.Items[4], "技能")
	if m.Items[5].Role != menu.WindowMenuRole {
		t.Fatalf("expected sixth top-level item to be WindowMenuRole, got role=%v", m.Items[5].Role)
	}
	requireTopLevelLabel(t, m.Items[6], "帮助")
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

	if got := len(m.Items); got != 5 {
		t.Fatalf("expected 5 top-level items for windows, got %d", got)
	}

	requireTopLevelLabel(t, m.Items[0], "文件")
	requireTopLevelLabel(t, m.Items[1], "编辑")
	if m.Items[1].Role == menu.EditMenuRole {
		t.Fatalf("expected second top-level item NOT to be EditMenuRole on windows")
	}
	requireTopLevelLabel(t, m.Items[2], "模板")
	requireTopLevelLabel(t, m.Items[3], "技能")
	requireTopLevelLabel(t, m.Items[4], "帮助")

	if hasTopLevelRole(m, menu.AppMenuRole) {
		t.Fatal("windows menu should not include AppMenuRole")
	}
	if hasTopLevelRole(m, menu.WindowMenuRole) {
		t.Fatal("windows menu should not include WindowMenuRole")
	}

	assertSharedFileMenu(t, m)
}

func TestBuildMenu_WindowsEditMenu(t *testing.T) {
	m := buildMenuForPlatform(&App{}, "windows")
	editMenu := findSubmenu(m, "编辑")
	if editMenu == nil {
		t.Fatal("编辑 menu not found")
	}

	// 8 items: 6 text + 2 separators
	if got := len(editMenu.Items); got != 8 {
		t.Fatalf("编辑 menu: expected 8 items, got %d", got)
	}

	expectedLabels := []string{"撤销", "重做", "剪切", "复制", "粘贴", "全选"}
	textOnly := textItems(editMenu)
	if got := len(textOnly); got != len(expectedLabels) {
		t.Fatalf("编辑 menu text items: expected %d, got %d", len(expectedLabels), got)
	}
	for i, expected := range expectedLabels {
		if textOnly[i].Label != expected {
			t.Errorf("编辑 menu text item[%d]: expected %q, got %q", i, expected, textOnly[i].Label)
		}
	}

	// Check expected accelerators
	accelTests := []struct {
		label string
		accel *keys.Accelerator
	}{
		{"撤销", keys.CmdOrCtrl("z")},
		{"重做", keys.Combo("z", keys.CmdOrCtrlKey, keys.ShiftKey)},
		{"剪切", keys.CmdOrCtrl("x")},
		{"复制", keys.CmdOrCtrl("c")},
		{"粘贴", keys.CmdOrCtrl("v")},
		{"全选", keys.CmdOrCtrl("a")},
	}

	for _, tt := range accelTests {
		t.Run(tt.label, func(t *testing.T) {
			item := findItem(editMenu, tt.label)
			if item == nil {
				t.Fatalf("item %q not found in 编辑 menu", tt.label)
			}
			if !accelEqual(item.Accelerator, tt.accel) {
				t.Errorf("item %q: accelerator mismatch\n  got:  %+v\n  want: %+v",
					tt.label, item.Accelerator, tt.accel)
			}
		})
	}

	// Verify separators at positions 2 and 7
	if editMenu.Items[2].Type != menu.SeparatorType {
		t.Errorf("编辑 menu[2]: expected separator, got type=%v", editMenu.Items[2].Type)
	}
	if editMenu.Items[6].Type != menu.SeparatorType {
		t.Errorf("编辑 menu[6]: expected separator, got type=%v", editMenu.Items[6].Type)
	}
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

func TestBuildMenu_TemplateSubmenu(t *testing.T) {
	m := buildMenuForPlatform(&App{}, "darwin")
	tplMenu := findSubmenu(m, "模板")
	if tplMenu == nil {
		t.Fatal("模板 menu not found")
	}

	items := textItems(tplMenu)
	if got := len(items); got != 1 {
		t.Fatalf("模板 menu: expected 1 item, got %d", got)
	}

	store := findItem(tplMenu, "模板商店")
	if store == nil {
		t.Fatal("模板商店 not found in 模板 menu")
	}
	expected := keys.Combo("t", keys.CmdOrCtrlKey, keys.ShiftKey)
	if !accelEqual(store.Accelerator, expected) {
		t.Errorf("模板商店: accelerator mismatch, got %+v, want %+v", store.Accelerator, expected)
	}

	if item := findItem(tplMenu, "模板管理…"); item != nil {
		t.Fatal("模板管理… should not exist in 模板 menu")
	}
}

func TestBuildMenu_SkillSubmenu(t *testing.T) {
	m := buildMenuForPlatform(&App{}, "darwin")
	skillMenu := findSubmenu(m, "技能")
	if skillMenu == nil {
		t.Fatal("技能 menu not found")
	}

	items := textItems(skillMenu)
	if got := len(items); got != 1 {
		t.Fatalf("技能 menu: expected 1 item, got %d", got)
	}

	store := findItem(skillMenu, "技能商店")
	if store == nil {
		t.Fatal("技能商店 not found in 技能 menu")
	}
	expected := keys.Combo("k", keys.CmdOrCtrlKey, keys.ShiftKey)
	if !accelEqual(store.Accelerator, expected) {
		t.Errorf("技能商店: accelerator mismatch, got %+v, want %+v", store.Accelerator, expected)
	}

	if item := findItem(skillMenu, "技能管理…"); item != nil {
		t.Fatal("技能管理… should not exist in 技能 menu")
	}
}
