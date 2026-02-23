# 提示词：Rust Starter Windows 构建修复 + CONVENTIONS.md 集中管理

## 问题 1：Rust Starter 的 Windows ARM64 构建

**仓库**：`Presto-io/presto-template-starter-rust`

**现状**：`.github/workflows/release.yml` 中有 `aarch64-pc-windows-msvc` 目标，但：
- Windows ARM64 MSVC 交叉编译需要特殊的链接器配置
- CI 用的 `windows-latest` runner 是 x86_64，可能无法直接交叉编译到 ARM64
- 对比 Go starter 和 TS starter 都能简单地交叉编译，Rust 这里需要额外处理

**修复方案**（二选一）：

### 方案 A：使用 cross（推荐）

在 release.yml 中，为 Windows ARM64 也使用 `cross`（目前只有 Linux ARM64 用了 cross）：

```yaml
- name: Install cross (for cross-compilation)
  if: matrix.target == 'aarch64-unknown-linux-gnu' || matrix.target == 'aarch64-pc-windows-msvc'
  run: cargo install cross --locked
```

并在 build step 中对应修改：

```yaml
- name: Build
  run: |
    if [[ "${{ matrix.target }}" == "aarch64-unknown-linux-gnu" || "${{ matrix.target }}" == "aarch64-pc-windows-msvc" ]]; then
      cross build --release --target ${{ matrix.target }}
    else
      cargo build --release --target ${{ matrix.target }}
    fi
```

### 方案 B：移除 Windows ARM64 目标

如果 Windows ARM64 用户很少，可以暂时不支持：

```yaml
matrix:
  include:
    - { target: x86_64-apple-darwin, runner: macos-latest, suffix: darwin-amd64 }
    - { target: aarch64-apple-darwin, runner: macos-latest, suffix: darwin-arm64 }
    - { target: x86_64-unknown-linux-gnu, runner: ubuntu-latest, suffix: linux-amd64 }
    - { target: aarch64-unknown-linux-gnu, runner: ubuntu-latest, suffix: linux-arm64 }
    - { target: x86_64-pc-windows-msvc, runner: windows-latest, suffix: windows-amd64.exe }
    # aarch64-pc-windows-msvc 暂不支持
```

**建议先用方案 A 试试，不行再降级到方案 B。**

---

## 问题 2：CONVENTIONS.md 集中管理

**现状**：三个 starter 仓库各自有一份 CONVENTIONS.md 副本，内容可能不同步。

**解决方案**：将 CONVENTIONS.md 的权威版本放在 Presto-Homepage 仓库，三个 starter 改为链接。

### 步骤

1. **在 Presto-Homepage 仓库**（`Presto-io/Presto-Homepage`）中：
   - 创建 `docs/conventions.md`（从任一 starter 的 CONVENTIONS.md 复制最新版本）
   - 确保该文件可通过 GitHub Pages 访问（如果 Homepage 部署了的话）

2. **在三个 starter 仓库中**：
   - 将 CONVENTIONS.md 内容替换为：

```markdown
# Template Development Conventions

本文档已迁移至中心位置，请访问：

**https://github.com/Presto-io/Presto-Homepage/blob/main/docs/conventions.md**

（或 Presto 官网文档页面，如果已部署）

此文件保留是为了让 AI 工具能找到它。完整开发规范请点击上方链接。
```

   - 同时更新 CLAUDE.md 和 AGENTS.md 中的引用路径

3. **可选：CI 自动同步**
   - 在 Presto-Homepage 仓库添加一个 GitHub Action，当 `docs/conventions.md` 更新时，自动向三个 starter 仓库提 PR 更新 CONVENTIONS.md
   - 这是可选的，手动同步也可以接受（因为 conventions 不会频繁变更）

### Commit 消息

```
docs: CONVENTIONS.md 迁移至 Presto-Homepage 集中管理
```
