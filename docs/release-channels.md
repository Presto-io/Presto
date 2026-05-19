# Release Channels

Presto 发布同一版本时提供默认精剪包和离线便携包。两个渠道共用核心应用代码，但资源内置、在线能力和升级方式不同。

## 默认精剪包

默认精剪包继续使用既有文件名，不增加 `slim` 后缀：

- `Presto-{version}-macOS-{arch}.dmg`
- `Presto-{version}-windows-{arch}-installer.exe`
- `Presto-{version}-linux-{arch}.tar.gz`

这是普通用户默认下载的包。它保留在线 registry、首启模板 bootstrap、模板自动更新、在线模板商店、在线技能商店和 GitHub Release 更新检查能力。

## 离线便携包

离线便携包的文件名包含 `portable`，例如：

- `Presto-{version}-portable-macOS-{arch}.dmg`
- `Presto-{version}-portable-windows-amd64.zip`
- `Presto-{version}-portable-linux-amd64.AppImage`

离线便携包内置 Typst、Tinymist 和全部官方模板，包括 `gongwen`、`jiaoan-shicao`、`jiaoan-jihua`。它的渠道契约是严格离线：应用进程不得发起外部网络请求，在线模板商店、在线技能商店、registry refresh、首启默认模板下载、模板自动更新、设置页更新检查和 GitHub Release 更新请求都必须隐藏或禁用。

允许的通信只限本机回环，例如 `localhost`、`127.0.0.1` 和 `::1`。离线便携包不支持在线自更新；升级方式是安装新版 portable 包。

Windows portable 当前发布为明确的 ZIP fallback，包内包含应用、Typst、Tinymist 和官方模板资源。单文件 `.exe` 便携包仍是目标形态，但当前 release 产物不要写成已经实现单文件 `.exe`。

## 模板覆盖层

离线便携包内置的官方模板是只读基线，随应用包发布。拖入官方模板 ZIP 时，Presto 会把模板写入用户数据目录作为覆盖层，而不是修改应用包、`.app` bundle、安装目录或 ZIP fallback 内部资源。

同名模板优先使用用户数据目录中的覆盖版本。删除用户覆盖版本后，Presto 会恢复使用内置只读基线。ZIP 覆盖只用于官方模板，不用于更新 Typst 或 Tinymist 运行件。

## Linux

Linux portable 优先发布 AppImage，因为它最符合 Linux GUI 便携应用的单文件使用预期，并能携带运行件和官方模板资源。

如果 CI 或平台条件暂时无法生成 AppImage，可以发布 fallback tar 包，但必须在 release notes、operator 记录或 smoke 记录中明确标记为 fallback，并说明原因。fallback tar 不能静默等同于 AppImage。
