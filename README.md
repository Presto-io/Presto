# Presto

Markdown → Typst → PDF，一站式文档转换平台。

[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go)](https://go.dev)
[![Svelte](https://img.shields.io/badge/Svelte-5-FF3E00?logo=svelte)](https://svelte.dev)
[![Wails](https://img.shields.io/badge/Wails-2.11-412991)](https://wails.io)
[![Typst](https://img.shields.io/badge/Typst-0.14-239DAD)](https://typst.app)
[![License](https://img.shields.io/badge/License-MIT-green)](LICENSE)

用 Markdown 写内容，选一个模板，Presto 帮你排版成专业 PDF。不用学 LaTeX，不用折腾 Word 样式。

<!-- 截图占位：替换为实际应用截图 -->
<!-- ![Presto 编辑器界面](docs/screenshots/editor.png) -->

## 功能特性

- 实时预览：编辑 Markdown，即时看到排版效果（SVG 渲染，支持多页）
- 编辑器与预览双向滚动同步
- CodeMirror 6 编辑器，支持语法高亮、中文搜索、自动换行
- 模板系统：插件化架构，模板是独立可执行文件，支持从 GitHub 安装第三方模板
- 批量转换：一次处理多个文件
- 桌面端：原生 macOS 菜单栏、文件对话框、键盘快捷键（Cmd+O 打开、Cmd+E 导出、Cmd+, 设置）
- Web 端：Docker 一键部署，浏览器直接使用
- 跨平台打包：macOS（Universal）、Windows、Linux

## 内置模板

| 模板 | 用途 | 说明 |
|------|------|------|
| `gongwen` | 公文排版 | 符合 GB/T 9704-2012 标准，支持方正小标宋、仿宋等字体 |
| `jiaoan-shicao` | 教案试操 | 教学计划表格格式，自动编号和单元格合并 |

## 快速开始

### 前置依赖

- [Go 1.25+](https://go.dev/dl/)
- [Node.js 22+](https://nodejs.org/)
- [Typst 0.14+](https://github.com/typst/typst/releases)
- [Wails CLI](https://wails.io/docs/gettingstarted/installation)（仅桌面端需要）

### macOS 桌面端

```bash
git clone https://github.com/mrered/presto.git
cd presto
make desktop
make run-desktop
```

### Docker 部署（Web 端）

```bash
docker compose up -d
```

浏览器打开 `http://localhost:8080`。

也可以手动构建镜像：

```bash
docker build -t presto .
docker run -d -p 8080:8080 -v presto-data:/root/.presto presto
```

### 从源码运行

```bash
# 构建前端
make frontend

# 构建并安装模板
make templates
make install-templates

# 启动服务器
make dev
```

## 使用指南

### 基本工作流

1. 在左侧编辑器中编写 Markdown 内容
2. 从顶部下拉菜单选择模板
3. 右侧实时预览排版效果
4. 点击导出按钮（或 Cmd+E）下载 PDF

### YAML Front Matter

模板通过 YAML front matter 接收元数据。以 `gongwen` 模板为例：

```markdown
---
title: 关于开展安全检查的通知
author: 办公室
date: 2026年2月17日
signature: 某某单位
---

正文内容...
```

不同模板支持的字段不同，具体参考模板的 `manifest.json` 中的 `frontmatterSchema`。

### 插入图片

Markdown 标准图片语法：

```markdown
![图片描述](images/fig.png)
```

路径解析规则：
- 通过 Cmd+O 打开文件后，图片路径相对于文件所在目录解析。比如文件在 `~/Documents/report/` 下，`![](images/fig.png)` 会找 `~/Documents/report/images/fig.png`
- 直接在编辑器中输入内容（未打开文件）时，使用绝对路径：`![](/Users/me/images/fig.png)`

### 键盘快捷键

| 快捷键 | 功能 |
|--------|------|
| `Cmd+O` | 打开 Markdown 文件 |
| `Cmd+E` | 导出 PDF |
| `Cmd+,` | 打开设置 |
| `Cmd+F` | 编辑器内搜索 |

## 模板系统

Presto 的模板是独立的可执行文件，通过 stdin/stdout 协议与主程序通信：

```
Markdown (stdin) → 模板二进制 → Typst 源码 (stdout)
```

### 模板结构

```
~/.presto/templates/my-template/
├── manifest.json           # 模板元数据
└── presto-template-my-template  # 可执行文件
```

### manifest.json 格式

```json
{
  "name": "my-template",
  "displayName": "我的模板",
  "description": "模板描述",
  "version": "1.0.0",
  "author": "作者",
  "license": "MIT",
  "minPrestoVersion": "0.1.0",
  "frontmatterSchema": {
    "title": { "type": "string", "required": true },
    "author": { "type": "string" }
  }
}
```

### 从 GitHub 安装模板

Presto 通过 GitHub Search API 发现带有 `presto-template` topic 的仓库。在模板商店页面可以浏览和一键安装。

安装流程：搜索 GitHub → 下载对应平台的 Release 二进制 → 解压到 `~/.presto/templates/`。

### 开发自定义模板

1. 创建一个 Go（或任意语言）程序，从 stdin 读取 Markdown，向 stdout 输出 Typst 源码
2. 支持 `--manifest` 参数，输出 JSON 格式的模板元数据
3. 编写 `manifest.json`
4. 在 GitHub 仓库添加 `presto-template` topic
5. 创建 Release，上传各平台二进制文件（命名格式：`presto-template-{name}-{os}-{arch}`）

## 架构概览

```
┌─────────────────────────────────────────────────┐
│                   前端 (SvelteKit 2)             │
│  ┌──────────┐  ┌──────────┐  ┌───────────────┐  │
│  │ Editor   │  │ Preview  │  │ TemplateStore │  │
│  │(CodeMirror)│ │ (SVG)   │  │  (GitHub API) │  │
│  └────┬─────┘  └────┬─────┘  └───────┬───────┘  │
│       └──────────────┴────────────────┘          │
│                      │ fetch                     │
├──────────────────────┼───────────────────────────┤
│                Go HTTP API                       │
│  ┌──────────┐  ┌──────────┐  ┌───────────────┐  │
│  │ convert  │  │ compile  │  │  templates    │  │
│  │ handler  │  │ handler  │  │  handler      │  │
│  └────┬─────┘  └────┬─────┘  └───────┬───────┘  │
│       │              │                │          │
│  ┌────┴─────┐  ┌────┴─────┐  ┌───────┴───────┐  │
│  │ Template │  │  Typst   │  │   Template    │  │
│  │ Executor │  │ Compiler │  │   Manager     │  │
│  └──────────┘  └──────────┘  └───────────────┘  │
└─────────────────────────────────────────────────┘
```

### 技术栈

| 层 | 技术 |
|----|------|
| 前端框架 | SvelteKit 2 + Svelte 5（runes 语法） |
| 编辑器 | CodeMirror 6 + Markdown 扩展 |
| 图标 | Lucide Svelte |
| 后端 | Go 标准库 `net/http` |
| 桌面框架 | Wails v2.11 |
| 排版引擎 | Typst CLI |
| Markdown 解析 | Goldmark（模板内部使用） |
| 容器化 | Docker 多阶段构建 |

### 数据流

1. 用户在编辑器输入 Markdown
2. 前端调用 `POST /api/convert`，发送 Markdown + 模板 ID
3. 后端通过 Template Executor 调用模板二进制，将 Markdown 转为 Typst 源码
4. 前端调用 `POST /api/compile-svg`，发送 Typst 源码
5. 后端调用 Typst CLI 编译为 SVG，返回各页 SVG 数据
6. 前端渲染 SVG 页面到预览区域
7. 导出时调用 `POST /api/compile`，获取 PDF 二进制并下载

## API 参考

所有接口前缀为 `/api`，请求和响应均为 JSON（除编译接口外）。

### 转换与编译

| 方法 | 路径 | 说明 | 请求体 | 响应 |
|------|------|------|--------|------|
| POST | `/api/convert` | Markdown → Typst | `{"markdown": "...", "templateId": "..."}` | `{"typst": "..."}` |
| POST | `/api/compile` | Typst → PDF | `text/plain`（Typst 源码） | `application/pdf` |
| POST | `/api/compile-svg` | Typst → SVG | `text/plain`（Typst 源码） | `{"pages": ["<svg>...", ...]}` |
| POST | `/api/convert-and-compile` | Markdown → PDF | `{"markdown": "...", "templateId": "..."}` | `application/pdf` |
| POST | `/api/batch` | 批量转换 | 待实现 | — |

### 模板管理

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/templates` | 列出已安装模板 |
| GET | `/api/templates/discover` | 从 GitHub 搜索可用模板 |
| POST | `/api/templates/{id}/install` | 安装模板 |
| DELETE | `/api/templates/{id}` | 卸载模板 |
| GET | `/api/templates/{id}/manifest` | 获取模板元数据 |

### 健康检查

```
GET /api/health → {"status": "ok"}
```

## 构建与打包

项目使用 Makefile 管理构建流程。

### 常用命令

```bash
make frontend          # 构建前端
make server            # 构建 HTTP 服务器
make desktop           # 构建桌面端（Wails）
make templates         # 构建模板二进制
make install-templates # 安装模板到 ~/.presto/templates/
make dev               # 开发模式运行服务器
make run-desktop       # 构建并运行桌面端
make clean             # 清理构建产物
```

### 跨平台打包

```bash
# macOS
make dist-macos-arm64      # Apple Silicon .app
make dist-macos-amd64      # Intel .app
make dist-macos-universal  # Universal .app
make dist-dmg-arm64        # Apple Silicon DMG
make dist-dmg-amd64        # Intel DMG
make dist-dmg-universal    # Universal DMG

# Windows（需要 mingw-w64）
make dist-windows-amd64

# Linux（通过 Docker 构建）
make dist-linux-amd64

# 全平台
make dist
```

打包产物输出到 `dist/` 目录。macOS .app 会自动捆绑 Typst 二进制到 `Contents/Resources/`。

### 项目结构

```
presto/
├── cmd/
│   ├── presto-desktop/    # Wails 桌面端入口
│   ├── presto-server/     # HTTP 服务器入口
│   ├── gongwen/           # 公文模板
│   └── jiaoan-shicao/     # 教案试操模板
├── internal/
│   ├── api/               # HTTP 路由和处理器
│   ├── template/          # 模板管理、执行、GitHub 集成
│   └── typst/             # Typst CLI 封装
├── frontend/              # SvelteKit 前端
│   └── src/
│       ├── lib/
│       │   ├── api/       # API 客户端
│       │   └── components/# Editor, Preview, TemplateSelector
│       └── routes/        # 页面路由
├── packaging/             # 平台打包配置
├── Dockerfile
├── docker-compose.yml
└── Makefile
```

## 贡献指南

欢迎提交 Issue 和 Pull Request。

### 开发环境搭建

```bash
git clone https://github.com/mrered/presto.git
cd presto
make templates && make install-templates
cd frontend && npm install && cd ..
make dev
```

### 提交模板

如果你开发了新模板，欢迎分享：

1. 在 GitHub 创建仓库，添加 `presto-template` topic
2. 按照模板协议实现 stdin/stdout 接口
3. 创建 Release 并上传各平台二进制
4. 社区用户就能在模板商店中发现并安装你的模板

### 代码规范

- Go 代码遵循标准 `gofmt` 格式
- 前端使用 TypeScript，遵循 SvelteKit 约定
- 提交信息使用 `feat:` / `fix:` / `docs:` 等前缀

## 开源协议

[MIT License](LICENSE)

## 致谢

- [Typst](https://typst.app) — 现代排版引擎
- [Wails](https://wails.io) — Go 桌面应用框架
- [SvelteKit](https://kit.svelte.dev) — 前端框架
- [CodeMirror](https://codemirror.net) — 代码编辑器
- [Goldmark](https://github.com/yuin/goldmark) — Go Markdown 解析器
- [Lucide](https://lucide.dev) — 图标库
