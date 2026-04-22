# Presto — AI 开发指南

> 组织级规则见 `../Presto-homepage/docs/ai-guide.md`
> 详细架构、构建命令、项目结构、测试方法、环境依赖 → 见 `docs/ai-reference.md`
> 整体架构设计 → 见 `../Presto-homepage/docs/specs/Presto-architecture.md`

Presto: Markdown → Typst → PDF 文档转换平台（桌面端 Wails + Web 端 Docker）。模块 `github.com/mrered/presto`。

## 代码规范

### Go

- `gofmt` 格式，错误返回 `error` 不 panic，日志 `log.Printf("[module] ...")`
- 安全标注用 `// SEC-XX:` 注释

### 前端 (Svelte 5 + TypeScript)

- Svelte 5 runes (`$state`, `$derived`, `$effect`)，TypeScript strict
- 组件 `frontend/src/lib/components/`，API `frontend/src/lib/api/`，状态 `.svelte.ts`

## 安全

代码中 `SEC-XX` 标注的安全措施**不能降级**（SEC-02/09/12/14/19/21/25）。详见 `docs/security-audit.md`。

## 禁止事项

- **不要**修改 `.github/workflows/release.yml` 除非被明确要求
  （2026-04-23: 已授权将 GO_VERSION 从 1.25.8 更新到 1.25.9 以对齐 go.mod）
- **不要**直接修改 `cmd/presto-desktop/build/` 中的内容（前端构建产物）
- **不要**引入新的第三方 Go 依赖，除非明确讨论过
