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
docker run -d -p 8080:8080 -v presto-data:/home/presto/.presto ghcr.io/presto-io/presto
```

浏览器打开 `http://localhost:8080`。

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
make check             # 推荐的统一质量入口
make check-go          # Go 测试 + vet
make check-frontend    # 前端检查 + 构建
make desktop           # 本地桌面端构建（不属于默认 CI 基线）
```

## 开源协议

[MIT License](LICENSE)
