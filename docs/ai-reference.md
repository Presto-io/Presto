# Presto — AI 详细参考

> 本文件包含项目的详细技术参考信息，供需要时查阅。
> 核心规则和约束见 `CLAUDE.md`。

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

所有构建和基础检查都以 `Presto/` 仓库根目录为执行基准：

```bash
make check                 # 推荐的统一质量入口（必需基线）
make check-go              # Go baseline: go test ./... + go vet ./...
make check-frontend        # Frontend baseline: npm run check + npm run build
make check-go-race         # Go 扩展检查: go test ./... -race
make check-desktop-compile # 本地桌面编译验证: go build ./cmd/presto-desktop
make check-local           # 开发者本地聚合: check + check-go-race + check-desktop-compile
make frontend              # 构建前端 (cd frontend && npm run build)
make server                # 构建 HTTP 服务器 → bin/presto-server
make desktop               # 构建桌面端 → bin/presto-desktop
make dev                   # 开发模式运行服务器 (go run ./cmd/presto-server/)
make run-desktop           # 构建并运行桌面端
make clean                 # 清理构建产物
```

说明：

- `make check` 是当前推荐的本地 / CI 共用基础入口。
- `make check-local` 是开发者本地扩展套件，包含基线 + race 检测 + 桌面编译验证。
- `make check-go-race` 和 `make check-desktop-compile` 属于扩展本地检查，不应直接复制到 CI 基线作业。
- `make desktop`、`dist-*` 等桌面或平台特定构建不属于当前默认自动化基线。

### 跨平台打包

```bash
make dist-macos-arm64      # Apple Silicon .app
make dist-macos-universal  # Universal .app
make dist-dmg-arm64        # Apple Silicon DMG
make dist-windows-amd64    # Windows EXE
make inno                  # Windows 安装器（Inno Setup）
make dist-linux-amd64      # Linux 二进制
make dist                  # 全平台打包
```

## 测试

```bash
make check                 # 推荐先跑的完整基线（必需）
make check-go              # Go baseline
make check-go-race         # Go race 检测（扩展，较慢）
make check-desktop-compile # 本地桌面编译验证（平台相关）
make check-local           # 开发者本地聚合：check + race + desktop compile
go test ./internal/...     # 运行内部 Go 单元测试
go test ./internal/template/...  # 模板相关测试
go test ./internal/typst/...     # Typst 编译器测试
go vet ./...               # Go 静态检查
```

前端检查：

```bash
make check-frontend             # 推荐入口
cd frontend && npm run check     # svelte-check + TypeScript
cd frontend && npm run build     # 生产构建验证
```

**分类：**

- **必需基线：** `make check`（含 `check-go` + `check-frontend`）
- **扩展本地：** `make check-go-race`（race 检测）、`make check-desktop-compile`（桌面编译）
- **开发者聚合：** `make check-local`（基线 + 扩展，仅供本地使用）
- 扩展本地目标不应直接复制到 CI 基线作业，CI 对齐由 Phase 11 负责

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

### 安全策略详情

项目已完成安全加固，代码中有 `SEC-XX` 标注：
- SEC-02: Typst 编译器使用受限 root 目录
- SEC-09: API Key 认证
- SEC-12: 编译超时限制 (60s)
- SEC-14: 默认绑定 localhost
- SEC-19: 速率限制
- SEC-21: Docker 非 root 用户
- SEC-25: 临时文件用随机后缀防竞争

完整安全审计：见 `docs/security-audit.md`

## 环境依赖

开发环境需要：
- Go 1.25+
- Node.js 22+ / npm
- Typst 0.14+（`brew install typst` 或手动安装）
- Wails CLI（仅桌面端开发：`go install github.com/wailsapp/wails/v2/cmd/wails@latest`）
