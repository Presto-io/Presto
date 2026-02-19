# Presto — AI 开发指南

## 项目简介

Presto 是一个 Markdown → Typst → PDF 一站式文档转换平台，支持桌面端（Wails）和 Web 端（Docker）。

- **Go 模块**: `github.com/mrered/presto`
- **版本**: 0.1.0
- **协议**: MIT

## 技术栈

| 层 | 技术 | 版本 |
|---|---|---|
| 后端 | Go 标准库 `net/http` | Go 1.25 |
| 桌面框架 | Wails | v2.11 |
| 前端框架 | SvelteKit 2 + Svelte 5 (runes 语法) | Svelte 5 |
| 编辑器 | CodeMirror 6 | 6.x |
| 类型系统 | TypeScript (strict) | 5.x |
| 构建工具 | Vite | 7.x |
| 排版引擎 | Typst CLI | 0.14.2 |
| 容器化 | Docker 多阶段构建 | — |

## 项目结构

```
presto/
├── cmd/
│   ├── presto-desktop/    # Wails 桌面端入口 (main.go + embed 前端)
│   ├── presto-server/     # HTTP 服务器入口
│   ├── gongwen/           # 公文模板二进制
│   └── jiaoan-shicao/     # 教案试操模板二进制
├── internal/
│   ├── api/               # HTTP 路由和处理器 (server.go, convert.go, templates.go, middleware.go)
│   ├── template/          # 模板管理、执行器、GitHub 集成、内置模板
│   └── typst/             # Typst CLI 封装 (compiler.go)
├── frontend/              # SvelteKit 前端 (独立 npm 项目)
│   └── src/
│       ├── lib/
│       │   ├── api/       # API 客户端 (client.ts, types.ts)
│       │   ├── components/# Editor, Preview, TemplateSelector
│       │   └── stores/    # Svelte 状态管理 (editor.svelte.ts)
│       └── routes/        # 页面: 主页、批量、设置
├── packaging/             # macOS 打包资源 (Info.plist, icon, DMG 配置)
├── test/                  # 测试产物 (SVG 输出)
├── docs/                  # 设计文档、重构计划
├── Makefile               # 构建系统入口
├── Dockerfile             # Web 端多阶段构建
└── docker-compose.yml     # 一键部署
```

## 构建命令

所有构建通过 Makefile 管理：

```bash
make frontend              # 构建前端 (cd frontend && npm run build)
make server                # 构建 HTTP 服务器 → bin/presto-server
make desktop               # 构建桌面端 → bin/presto-desktop
make templates             # 构建模板二进制 → bin/presto-template-*
make install-templates     # 安装模板到 ~/.presto/templates/
make dev                   # 开发模式运行服务器 (go run ./cmd/presto-server/)
make run-desktop           # 构建并运行桌面端
make clean                 # 清理构建产物
```

### 跨平台打包

```bash
make dist-macos-arm64      # Apple Silicon .app
make dist-macos-universal  # Universal .app
make dist-dmg-arm64        # Apple Silicon DMG
make dist-windows-amd64    # Windows EXE
make dist-linux-amd64      # Linux 二进制
make dist                  # 全平台打包
```

## 测试

```bash
go test ./internal/...     # 运行所有 Go 单元测试
go test ./internal/template/...  # 模板相关测试
go test ./internal/typst/...     # Typst 编译器测试
go vet ./...               # Go 静态检查
```

前端检查：

```bash
cd frontend && npm run check     # svelte-check + TypeScript
```

## 代码规范

### Go

- 遵循标准 `gofmt` 格式
- 错误处理：返回 `error`，不 panic（除 main 入口）
- 日志：使用 `log.Printf("[module] message")`，方括号标注模块名
- 包命名：短小写，如 `api`, `template`, `typst`
- 安全标注：关键安全措施用 `// SEC-XX:` 注释标注

### 前端 (Svelte 5 + TypeScript)

- 使用 Svelte 5 runes 语法 (`$state`, `$derived`, `$effect`)
- TypeScript strict 模式
- 组件放在 `frontend/src/lib/components/`
- API 客户端放在 `frontend/src/lib/api/`
- 状态管理用 `.svelte.ts` 文件 (runes store)

### Commit 规范

使用中文 commit 消息，格式：`<type>: <描述>`

类型前缀：
- `feat:` — 新功能
- `fix:` — Bug 修复
- `refactor:` — 重构
- `ui:` — UI 调整
- `sec:` — 安全相关
- `docs:` — 文档
- `merge:` — 分支合并

示例：`feat: 编辑器支持拖拽调节宽度比例`

## 架构要点

### 数据流

```
用户输入 Markdown
  → POST /api/convert (Markdown + templateId → Typst 源码)
  → POST /api/compile-svg (Typst → SVG 页面预览)
  → POST /api/compile (Typst → PDF 下载)
```

### 模板系统

模板是**独立的可执行文件**，通过 stdin/stdout 协议通信：
- 输入：stdin 接收 Markdown 文本
- 输出：stdout 输出 Typst 源码
- `--manifest` 参数：输出 JSON 元数据

**重要**：不要修改模板的 stdin/stdout 二进制协议。模板协议是稳定 API。

### 桌面端 vs Web 端

- **桌面端** (`presto-desktop`): Wails 嵌入前端资源，通过 Wails binding 直接调用 Go 方法
- **Web 端** (`presto-server`): 独立 HTTP 服务器，前端通过 fetch 调用 `/api/*` 端点
- 两者共享 `internal/` 层代码

### 安全策略

项目已完成安全加固，代码中有 `SEC-XX` 标注。修改时注意：
- SEC-02: Typst 编译器使用受限 root 目录
- SEC-09: API Key 认证
- SEC-12: 编译超时限制 (60s)
- SEC-14: 默认绑定 localhost
- SEC-19: 速率限制
- SEC-21: Docker 非 root 用户
- SEC-25: 临时文件用随机后缀防竞争

## 禁止事项

- **不要**修改模板二进制协议（stdin/stdout 接口）
- **不要**修改 `.github/workflows/release.yml` 除非被明确要求
- **不要**直接修改 `cmd/presto-desktop/build/` 中的内容（这是前端构建产物，由 `make frontend` 生成）
- **不要**引入新的第三方 Go 依赖，除非明确讨论过
- **不要**降低现有安全措施（SEC-XX 标注的功能）

## 环境依赖

开发环境需要：
- Go 1.25+
- Node.js 22+ / npm
- Typst 0.14+（`brew install typst` 或手动安装）
- Wails CLI（仅桌面端开发：`go install github.com/wailsapp/wails/v2/cmd/wails@latest`）

## 工作习惯

- **完成任务后必须立即 commit**：每完成一个逻辑任务，在回复用户之前就要 commit（按任务粒度，不是按文件）。不要等用户提醒，不要把 commit 留到最后
- Commit 消息用中文，格式见上方 Commit 规范
- 不要自动 push，除非被明确要求
- 当用户在对话中提出新的项目规范、架构决策或工作习惯要求时，应主动更新本文件（CLAUDE.md）以保持指南与实际一致
- 开始任务前先检查当前可用的技能（Skills），优先使用已安装的技能来完成工作，不要忽略它们直接蛮干
