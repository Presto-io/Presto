# Presto — AI 开发指南

Presto: Markdown → Typst → PDF 文档转换平台（桌面端 Wails + Web 端 Docker）。模块 `github.com/mrered/presto`。

> 详细架构、构建命令、项目结构、测试方法、环境依赖 → 见 `docs/ai-reference.md`
> 整体架构设计（模板系统、分发模型、商店、注册表等）→ 见 Presto-homepage 仓库 `docs/specs/Presto-architecture.md`

## 代码规范

### Go

- `gofmt` 格式，错误返回 `error` 不 panic，日志 `log.Printf("[module] ...")`
- 安全标注用 `// SEC-XX:` 注释

### 前端 (Svelte 5 + TypeScript)

- Svelte 5 runes (`$state`, `$derived`, `$effect`)，TypeScript strict
- 组件 `frontend/src/lib/components/`，API `frontend/src/lib/api/`，状态 `.svelte.ts`

### Commit

中文消息，格式 `<type>: <描述>`。类型：feat/fix/refactor/ui/sec/docs/merge

## 安全

代码中 `SEC-XX` 标注的安全措施**不能降级**（SEC-02/09/12/14/19/21/25）。详见 `docs/security-audit.md`。

## 禁止事项

- **不要**修改模板二进制协议（stdin/stdout 接口）
- **不要**修改 `.github/workflows/release.yml` 除非被明确要求
- **不要**直接修改 `cmd/presto-desktop/build/` 中的内容（前端构建产物）
- **不要**引入新的第三方 Go 依赖，除非明确讨论过
- **不要**降低现有安全措施

## 工作习惯

- **完成任务后必须立即 commit**：每完成一个逻辑任务，在回复用户之前就要 commit（按任务粒度，不是按文件）
- Commit 消息用中文，不要自动 push
- 当用户提出新规范/架构决策时，主动更新本文件
- 开始任务前先检查当前可用的技能（Skills），优先使用已安装的技能
