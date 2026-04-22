# Automation Baseline

**Last updated:** 2026-04-21
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

**注意：** 扩展本地目标（`check-go-race`、`check-desktop-compile`、`check-local`）不应被复制到 CI 面向的基线作业中，除非 Phase 11 明确做出此决定。

## CI gates

当前仓库已有的自动化门禁可按职责理解为两层：

### Shared baseline gate

- `Presto/.github/workflows/test.yml`
- 现在只负责运行共享基线：
  - `go test ./...`
  - `go vet ./...`
  - `cd frontend && npm run check`
  - `cd frontend && npm run build`
- 这个 workflow 已对齐到仓库根目录，不再依赖旧的 `cd Presto` 路径假设。

### Specialized gates

- `Presto/.github/workflows/security-scan.yml`
  - 负责 `govulncheck` 与 `npm audit`
  - 它反映的是真实依赖/安全问题，不应和路径漂移混为一类
- 桌面打包、发布、平台特定构建
  - 属于专门 workflow 的职责
  - 不应被塞回默认 `make check` 或基础 `test.yml`

当前状态总结：

- `test.yml` 已对齐到共享 baseline 入口。
- `security-scan.yml` 仍然是后续修复对象，因为失败来源是实际漏洞而非结构漂移。
- 桌面或平台特定构建仍属于后续 CI / release 收敛范围，不是当前基础门禁。

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

### Security Scan

- `Security Scan` recent run `24660047430` 失败属于真实依赖问题
- Go 侧日志指向 `crypto/x509@go1.25.8` 与 `crypto/tls@go1.25.8`，修复点在 `go1.25.9`
- npm audit 失败涉及 `@sveltejs/kit`、`devalue`、`dompurify`、`picomatch`、`vite`
- 这些问题应进入后续依赖/安全修复，而不是通过弱化 baseline 掩盖

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

- CI workflow / matrix / 安全门禁修复：后续 Phase 11
- carry-in 验证结论与里程碑收口：后续 Phase 12
