package main

import (
	"fmt"
	"runtime"

	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

func buildMenu(app *App) *menu.Menu {
	return buildMenuForPlatform(app, runtime.GOOS)
}

func buildMenuForPlatform(app *App, platform string) *menu.Menu {
	appMenu := menu.NewMenu()
	isDarwin := platform == "darwin"

	if isDarwin {
		appMenu.Append(menu.AppMenu())
	}

	addFileMenu(appMenu, app)
	if isDarwin {
		appMenu.Append(menu.EditMenu())
	} else {
		addEditMenu(appMenu, app)
	}
	addTemplateMenu(appMenu, app)
	addSkillMenu(appMenu, app)

	if isDarwin {
		appMenu.Append(menu.WindowMenu())
	}

	addHelpMenu(appMenu, app, !isDarwin)

	return appMenu
}


func addEditMenu(appMenu *menu.Menu, app *App) {
	editMenu := appMenu.AddSubmenu("编辑")
	editMenu.AddText("撤销", keys.CmdOrCtrl("z"), func(_ *menu.CallbackData) {
		wailsRuntime.EventsEmit(app.ctx, "menu:undo")
	})
	editMenu.AddText("重做", keys.Combo("z", keys.CmdOrCtrlKey, keys.ShiftKey), func(_ *menu.CallbackData) {
		wailsRuntime.EventsEmit(app.ctx, "menu:redo")
	})
	editMenu.AddSeparator()
	editMenu.AddText("剪切", keys.CmdOrCtrl("x"), func(_ *menu.CallbackData) {
		wailsRuntime.EventsEmit(app.ctx, "menu:cut")
	})
	editMenu.AddText("复制", keys.CmdOrCtrl("c"), func(_ *menu.CallbackData) {
		wailsRuntime.EventsEmit(app.ctx, "menu:copy")
	})
	editMenu.AddText("粘贴", keys.CmdOrCtrl("v"), func(_ *menu.CallbackData) {
		wailsRuntime.EventsEmit(app.ctx, "menu:paste")
	})
	editMenu.AddSeparator()
	editMenu.AddText("全选", keys.CmdOrCtrl("a"), func(_ *menu.CallbackData) {
		wailsRuntime.EventsEmit(app.ctx, "menu:selectall")
	})
}

func addFileMenu(appMenu *menu.Menu, app *App) {

	fileMenu := appMenu.AddSubmenu("文件")
	fileMenu.AddText("新建", keys.CmdOrCtrl("n"), func(_ *menu.CallbackData) {
		wailsRuntime.EventsEmit(app.ctx, "menu:new")
	})
	fileMenu.AddText("打开文件…", keys.CmdOrCtrl("o"), func(_ *menu.CallbackData) {
		wailsRuntime.EventsEmit(app.ctx, "menu:open")
	})
	app.saveMenuItem = fileMenu.AddText("保存", keys.CmdOrCtrl("s"), func(_ *menu.CallbackData) {
		wailsRuntime.EventsEmit(app.ctx, "menu:save")
	})
	app.saveMenuItem.Disabled = true
	fileMenu.AddText("另存为…", keys.Combo("s", keys.CmdOrCtrlKey, keys.ShiftKey), func(_ *menu.CallbackData) {
		wailsRuntime.EventsEmit(app.ctx, "menu:saveas")
	})
	app.exportMenuItem = fileMenu.AddText("导出 PDF…", keys.CmdOrCtrl("e"), func(_ *menu.CallbackData) {
		wailsRuntime.EventsEmit(app.ctx, "menu:export")
	})
	app.exportMenuItem.Disabled = true
	fileMenu.AddText("设置…", keys.CmdOrCtrl(","), func(_ *menu.CallbackData) {
		wailsRuntime.EventsEmit(app.ctx, "menu:settings")
	})
	fileMenu.AddSeparator()
	fileMenu.AddText("关闭窗口", keys.CmdOrCtrl("w"), func(_ *menu.CallbackData) {
		wailsRuntime.EventsEmit(app.ctx, "menu:close-window")
	})
}

func addTemplateMenu(appMenu *menu.Menu, app *App) {
	templateMenu := appMenu.AddSubmenu("模板")
	templateMenu.AddText("模板商店", keys.Combo("t", keys.CmdOrCtrlKey, keys.ShiftKey), func(_ *menu.CallbackData) {
		wailsRuntime.EventsEmit(app.ctx, "menu:store")
	})
}

func addSkillMenu(appMenu *menu.Menu, app *App) {
	skillMenu := appMenu.AddSubmenu("技能")
	skillMenu.AddText("技能商店", keys.Combo("k", keys.CmdOrCtrlKey, keys.ShiftKey), func(_ *menu.CallbackData) {
		wailsRuntime.EventsEmit(app.ctx, "menu:skill-store")
	})
}

func addHelpMenu(appMenu *menu.Menu, app *App, includeAbout bool) {
	helpMenu := appMenu.AddSubmenu("帮助")
	helpMenu.AddText("文档", nil, func(_ *menu.CallbackData) {
		wailsRuntime.BrowserOpenURL(app.ctx, "https://presto.io/docs")
	})
	if includeAbout {
		helpMenu.AddText("关于 Presto", nil, func(_ *menu.CallbackData) {
			app.ShowAboutDialog()
		})
	}
	helpMenu.AddText("检查更新", nil, func(_ *menu.CallbackData) {
		go app.CheckAndNotifyUpdate()
	})
}

func (a *App) ShowAboutDialog() {
	ver := a.GetVersion()
	wailsRuntime.MessageDialog(a.ctx, wailsRuntime.MessageDialogOptions{
		Type:    wailsRuntime.InfoDialog,
		Title:   "关于 Presto",
		Message: fmt.Sprintf("Presto %s\nMarkdown → Typst → PDF\n\n© 2024-2026 Presto", ver),
	})
}

func (a *App) UpdateMenuState(hasContent bool) {
	if a.saveMenuItem != nil {
		a.saveMenuItem.Disabled = !hasContent
	}
	if a.exportMenuItem != nil {
		a.exportMenuItem.Disabled = !hasContent
	}
	wailsRuntime.MenuUpdateApplicationMenu(a.ctx)
}
