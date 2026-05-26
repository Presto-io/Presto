# Presto

Markdown → Typst → PDF，一站式文档转换平台。

[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go)](https://go.dev)
[![Svelte](https://img.shields.io/badge/Svelte-5-FF3E00?logo=svelte)](https://svelte.dev)
[![Wails](https://img.shields.io/badge/Wails-2.11-412991)](https://wails.io)
[![Typst](https://img.shields.io/badge/Typst-0.14-239DAD)](https://typst.app)
[![License](https://img.shields.io/badge/License-MIT-green)](LICENSE)

用 Markdown 写内容，选一个模板，Presto 帮你排版成专业 PDF。不用学 LaTeX，不用折腾 Word 样式。

👉 **官网**：[https://presto.mre.red](https://presto.mre.red)

## 安装

### macOS（Homebrew）

```bash
brew install --cask brewforge/more/presto
```

### macOS / Windows / Linux（手动下载）

前往 [GitHub Releases](https://github.com/Presto-io/Presto/releases) 下载对应平台的安装包。

默认精剪包和离线便携包的差异、升级方式和发布资产规则见 [Release Channels](docs/release-channels.md)。

> **macOS 首次打开提示"无法验证开发者"？**
>
> 由于应用尚未签名，macOS 会阻止首次打开。请在终端运行：
>
> ```bash
> xattr -cr /Applications/Presto.app
> ```
>
> 或者右键点击 Presto.app → 打开 → 确认打开。

### Docker 部署（Web 端）

```bash
mkdir -p .presto-fonts
docker compose up -d
```

浏览器打开 `http://localhost:8080`。

默认的 `docker-compose.yml` 会持久化自定义字体目录：

- 字体文件放在 `./.presto-fonts`，对应容器内路径 `/data/fonts`。
- Presto 服务端启动时会从 `/data/fonts` 加载字体。

配置、模板、缓存和日志默认保存到 Docker 命名卷中，避免宿主机目录权限影响非 root 容器进程。如果你沿用旧版 `./.presto-data` bind mount，且看到 `/config` 或 `/data` permission denied，先修正旧目录权限：

```bash
sudo chown -R "$(id -u):$(id -g)" .presto-data
```

Presto 服务端默认从应用数据目录的 `fonts` 子目录加载字体；Docker 中是 `/data/fonts`。如果需要加载其他目录，可以通过 `FONT_PATHS` 环境变量覆盖，多个目录用冒号分隔，例如：

```yaml
services:
    presto:
        environment:
            FONT_PATHS: /data/fonts:/usr/share/fonts
```

## 平台支持

Presto 支持 Windows、macOS 和 Linux。详见 [平台说明](docs/platform-notes.md)。

## 文档

- **官网**：[https://presto.mre.red](https://presto.mre.red)
- **使用指南**、**模板开发**、**架构说明**等详细信息请访问官网

## 贡献指南

欢迎提交 Issue 和 Pull Request。

### 开发环境搭建

```bash
git clone https://github.com/Presto-io/Presto.git
cd Presto
npm install --prefix frontend
make check
make dev
```

常用本地命令：

```bash
make check             # 必需基线：提交 PR 前必须通过
make check-go          # Go 测试 + vet（make check 的子集）
make check-frontend    # 前端检查 + 构建（make check 的子集）
make check-local       # 开发者本地扩展套件（含 race 检测 + 桌面编译）
make desktop           # 本地桌面端构建（不属于默认 CI 基线）
```

> **提交 PR 前**至少运行 `make check`；高风险变更建议运行 `make check-local`。
> 扩展目标（`check-go-race`、`check-desktop-compile`）仅供本地使用，CI 对齐由 Phase 11 负责。

## 开源协议

[MIT License](LICENSE)
