# Automation Baseline

**Last updated:** 2026-04-22
**Scope:** `Presto` main repository

## Local baseline

当前团队应优先运行的统一入口：

```bash
make check
```

它会组合以下两层基线：

```bash
make check-go
make check-frontend
```

对应的实际命令是：

```bash
go test ./...
go vet ./...
cd frontend && npm run check
cd frontend && npm run build
```

当前口径：

- Go 基线要求通过且退出码为 `0`。
- 前端基线要求 `0 errors and 0 warnings`；Phase 10 已将原有 22 warnings 清零。
- `make check-go-race` 是已验证的扩展本地检查（`go test ./... -race`），但不在默认 `make check` 内。
- `make check-desktop-compile` 是本地平台相关构建验证（`go build ./cmd/presto-desktop`），不等于当前共享自动化基线。

### Extended local checks

开发者在提交高风险变更前，可运行扩展本地检查套件：

```bash
make check-local
```

它会组合以下三层：

```bash
make check            # 共享基线（必需）
make check-go-race    # Go race 检测（扩展）
make check-desktop-compile  # 桌面端编译验证（平台相关）
```

各目标分类：

| 目标 | 类型 | 说明 |
|------|------|------|
| `make check-go` | 必需基线 | `go test ./...` + `go vet ./...` |
| `make check-frontend` | 必需基线 | `npm run check` + `npm run build` |
| `make check-go-race` | 扩展本地 | `go test ./... -race`，更慢但可检测竞态 |
| `make check-desktop-compile` | 平台相关本地 | `go build ./cmd/presto-desktop`，依赖本地平台条件 |
| `make check-local` | 开发者聚合 | 上述全部，仅供本地使用，不应直接复制到 CI |

**注意：** 扩展本地目标（`check-go-race`、`check-desktop-compile`、`check-local`）不应被复制到 CI 面向的基线作业中，除非明确做出此决定。

## CI gates

当前仓库已有的自动化门禁可按职责理解为两层：

### Shared baseline gate

- `Presto/.github/workflows/test.yml`
- 使用 `make check` 作为单一命令（与本地基线单源对齐）
- `npm ci` 作为前置依赖安装步骤在 `make check` 之前运行
- `go-version-file: go.mod` 自动检测 Go 版本（当前 1.25.9）
- 三平台矩阵：ubuntu-latest / macos-latest / windows-latest
- `shell: bash` 确保 Windows 上 make 兼容

### Specialized gates

- `Presto/.github/workflows/security-scan.yml`
  - `go-vulncheck`：使用 `go-version-file: go.mod`（当前 Go 1.25.9），升级后 crypto 漏洞已修复
  - `npm audit`：降级为 informational only（`continue-on-error: true`），当前已知漏洞作为 follow-up 跟踪
  - 当前已知 npm 漏洞：dompurify（moderate）、picomatch（high）、vite（high）— 详见 Known follow-ups
- `Presto/.github/workflows/build-showcase.yml`
  - 已升级至 Node 22，与所有其他 workflow 一致

### D-05 resolution

Linux 桌面构建 CI 不加入 test.yml 基线。原因：

1. test.yml 基线不需要 webkit2gtk，三平台都能跑通
2. Linux 桌面构建依赖 `libwebkit2gtk`，在 ubuntu-latest 上脆弱且易变
3. release.yml 已在 ubuntu-22.04 runner 上验证 Linux 桌面构建
4. 桌面构建验证策略：release-only

### Known follow-ups

1. **release.yml Go 版本漂移**：`GO_VERSION: '1.25.8'` 硬编码，而 go.mod 已升级至 1.25.9。CLAUDE.md 禁止修改 release.yml。若限制解除，应更新此环境变量。

2. **npm audit 漏洞**：
   - `dompurify <=3.3.3` — mutation XSS, prototype pollution（moderate）
   - `picomatch 4.0.0-4.0.3` — ReDoS, method injection（high）
   - `vite 7.0.0-7.3.1` — path traversal, file read, fs.deny bypass（high）
   - 这些需要依赖升级，超出 Phase 11 范围

3. **Linux 桌面构建 CI**：不包含在 test.yml 基线中。桌面构建验证在 release.yml 的 ubuntu-22.04 runner 上进行。若 ubuntu-22.04 runner 被废弃，需要迁移至 webkit2gtk-4.1。

4. **Makefile Docker webkit2gtk**：`dist-linux-amd64` 仍使用 `libwebkit2gtk-4.0-dev`。在 Docker 中可用（基于 Debian），但长期应迁移至 4.1 以兼容 Ubuntu 24.04。

## Manual validation

以下项目当前不能被这份自动化基线完全替代：

- `v1.0.3` carry-in 的人工验证结论
- Phase 06 保存流程的人工 UAT / verification 收口
- 平台特定桌面体验确认
  - 例如真实设备上的菜单、安装、打包与运行行为

自动化基线的作用是先确保"共享质量入口"明确且可复现，不替代这些人工结论。

## Known noise / known failures

### Front-end warnings

- `npm run check` 当前报告 `0 errors and 0 warnings`（Phase 10 从 22 warnings 清零）
- `npm run build` 会输出 Vite chunk-size 提示（大模块 > 500 kB），属于构建优化建议，不是错误或 warning
- chunk-size 提示可通过 `build.chunkSizeWarningLimit` 配置或代码拆分消除，作为后续优化跟进

### Chunk-size build notes

- 主要来源：`editor-gongwen` 和 `editor-jiaoan` 页面内联了大量模板数据
- 当前不阻塞任何基线判断，但如果后续需要优化构建产物体积，这些页面是首要拆分目标

### Platform-specific build boundary

- `go build ./cmd/presto-desktop` 在当前 macOS 开发环境可通过
- 但它依赖的平台条件与共享 baseline 不同
- 因此桌面构建不等于当前 `make check` 基线

## Current operating rule

如果你的目标是判断"这个仓库当前是否满足共享自动化基线"，先运行：

```bash
make check
```

如果你想运行更完整的本地质量套件（含扩展检查），运行：

```bash
make check-local
```

如果你的目标是处理更重的质量面，再按问题类型继续：

- CI workflow / matrix / 安全门禁修复：Phase 11 已完成
- carry-in 验证结论与里程碑收口：后续 Phase 12
