# Presto 扩展生态规范

本文档是 Presto 扩展生态（模板、插件、Agent Skills）的技术规范，作为所有相关仓库的 single source of truth。

所有仓库的 AI 助手必须遵循本规范。

---

## 一、术语

| 术语 | 含义 |
|------|------|
| 扩展（Extension） | 模板、插件、Agent Skills 的统称 |
| 模板（Template） | 将 Markdown 转换为 Typst 排版源码的二进制工具 |
| 插件（Plugin） | 扩展 Presto 功能的组件（未来） |
| Agent Skill | AI 驱动的自动化技能（未来） |
| Registry | 扩展索引仓库，CI 自动构建静态 JSON + 预览资源 |
| Registry Deploy | 统一的 CDN 部署仓库，Cloudflare Pages 托管 |

---

## 二、二进制协议

模板编译后是一个**独立的可执行文件**，不限语言（Go、Rust、TypeScript/Bun、Python/PyInstaller 等均可），只要能编译为多平台二进制并遵守以下协议。

### 2.1 CLI 接口

| 调用方式 | 行为 |
|---------|------|
| `./binary` | stdin 读取 Markdown，stdout 输出 Typst 源码 |
| `./binary --manifest` | stdout 输出 manifest.json 内容 |
| `./binary --example` | stdout 输出 example.md 内容 |
| `./binary -o output.typ input.md` | 从文件读取，输出到文件（可选支持） |

### 2.2 安全约束

- Presto 以最小环境执行：`PATH=/usr/local/bin:/usr/bin:/bin`，无其他环境变量
- 执行超时：30 秒
- 二进制大小限制：100MB

### 2.3 嵌入数据

二进制必须内嵌以下数据（各语言的实现方式不同）：

| 语言 | manifest.json | example.md |
|------|--------------|------------|
| Go | `//go:embed manifest.json` | `//go:embed example.md` |
| Rust | `include_str!("manifest.json")` | `include_str!("example.md")` |
| TypeScript (Bun) | `Bun.file()` 编译时内嵌 | 同左 |
| Python | `importlib.resources` 或 `pkgutil` | 同左 |

---

## 三、manifest.json 规范

### 3.1 完整 Schema

```json
{
  "name": "gongwen",
  "displayName": "类公文模板",
  "description": "符合 GB/T 9704-2012 标准的类公文排版，支持标题、作者、日期、签名等元素",
  "version": "1.0.0",
  "author": "mrered",
  "license": "MIT",
  "category": "government",
  "keywords": ["公文", "国标", "GB/T 9704", "党政机关"],
  "minPrestoVersion": "0.1.0",
  "requiredFonts": [
    {
      "name": "FZXiaoBiaoSong-B05",
      "displayName": "方正小标宋",
      "url": "https://www.foundertype.com/...",
      "downloadUrl": null,
      "openSource": false
    }
  ],
  "frontmatterSchema": {
    "title": { "type": "string", "default": "请输入文字" },
    "author": { "type": "string", "default": "请输入文字" },
    "date": { "type": "string", "format": "YYYY-MM-DD" },
    "signature": { "type": "boolean", "default": false }
  }
}
```

### 3.2 字段说明

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `name` | string | 是 | 唯一标识符，kebab-case，如 `gongwen`、`jiaoan-shicao` |
| `displayName` | string | 是 | 显示名称（中文） |
| `description` | string | 是 | 简短描述（一句话） |
| `version` | string | 是 | 语义化版本号（semver） |
| `author` | string | 是 | 作者名或 GitHub 用户名 |
| `license` | string | 是 | SPDX 许可证标识符 |
| `category` | string | 是 | 分类，见下方枚举 |
| `keywords` | string[] | 是 | 搜索关键词（建议 2-6 个） |
| `minPrestoVersion` | string | 是 | 最低兼容 Presto 版本 |
| `requiredFonts` | FontRequirement[] | 否 | 所需字体列表 |
| `frontmatterSchema` | Record<string, FieldSchema> | 否 | 支持的 YAML frontmatter 字段 |

### 3.3 分类枚举

```
government   政务
education    教育（教案、考勤表、成绩册、比赛表等）
business     商务/办公
academic     学术
legal        法务
resume       简历/求职
creative     创意/设计
other        其他
```

空分类不显示——前端渲染时过滤掉没有扩展的分类。

### 3.4 FontRequirement

```json
{
  "name": "FZXiaoBiaoSong-B05",
  "displayName": "方正小标宋",
  "url": "https://www.foundertype.com/...",
  "downloadUrl": null,
  "openSource": false
}
```

- `url`：字体信息页（人工访问）
- `downloadUrl`：直链（开源字体才有，商业字体为 null）
- `openSource`：是否开源（决定能否自动下载）

### 3.5 FieldSchema

```json
{
  "type": "string",
  "default": "请输入文字",
  "format": "YYYY-MM-DD"
}
```

- `type`：`string` | `number` | `boolean` | `array`
- `default`：默认值（可选）
- `format`：格式提示（可选），如日期格式

---

## 四、平台矩阵

所有模板 Release 必须覆盖以下 6 个目标平台：

| OS | Arch | 二进制后缀 |
|----|------|-----------|
| darwin | arm64 | （无） |
| darwin | amd64 | （无） |
| linux | arm64 | （无） |
| linux | amd64 | （无） |
| windows | arm64 | `.exe` |
| windows | amd64 | `.exe` |

---

## 五、Release 资产命名规范

### 5.1 二进制命名

```
presto-template-{name}-{os}-{arch}[.exe]
```

示例：
```
presto-template-gongwen-darwin-arm64
presto-template-gongwen-darwin-amd64
presto-template-gongwen-linux-arm64
presto-template-gongwen-linux-amd64
presto-template-gongwen-windows-arm64.exe
presto-template-gongwen-windows-amd64.exe
```

### 5.2 校验文件

每个 Release 必须包含 `SHA256SUMS` 文件：

```
a1b2c3d4...  presto-template-gongwen-darwin-arm64
e5f6a7b8...  presto-template-gongwen-darwin-amd64
...
```

### 5.3 GitHub Topic

| 扩展类型 | Topic |
|---------|-------|
| 模板 | `presto-template` |
| 插件 | `presto-plugin` |
| Agent Skill | `presto-agent-skill` |

仓库必须设置对应 topic，registry CI 通过 topic 发现新扩展。

---

## 六、registry.json 规范

### 6.1 基础结构

```json
{
  "version": 1,
  "updatedAt": "2026-02-22T10:00:00Z",
  "categories": [
    { "id": "government", "label": { "zh": "政务", "en": "Government" } },
    { "id": "education", "label": { "zh": "教育", "en": "Education" } }
  ],
  "templates": [
    {
      "name": "gongwen",
      "displayName": "类公文模板",
      "description": "符合 GB/T 9704-2012 标准的类公文排版",
      "version": "1.0.0",
      "author": "mrered",
      "category": "government",
      "keywords": ["公文", "国标"],
      "license": "MIT",
      "trust": "official",
      "publishedAt": "2026-02-20T10:00:00Z",
      "repository": "https://github.com/Presto-io/official-templates"
    }
  ]
}
```

### 6.2 信任分级

| 级别 | 标识 | 条件 | 颜色 |
|------|------|------|------|
| `official` | 蓝色盾牌 | Presto-io 组织发布 | `#3b82f6` |
| `verified` | 绿色对勾 | 通过自动化审核 + 签名验证 | `#22c55e` |
| `community` | 灰色标签 | 仅收录，未审核 | `var(--color-muted)` |
| `unrecorded` | 警告标识 | 用户手动 URL 安装（不在 registry 中） | `var(--color-warning)` |

### 6.3 Registry 目录结构

每种扩展类型的 registry 仓库内部结构：

```
template-registry/
  templates/
    gongwen/
      manifest.json          ← ./binary --manifest 的输出
      README.md              ← 从模板仓库首页获取
      example.md             ← ./binary --example 的输出
      preview-1.svg          ← typst compile 生成
      preview-2.svg
    jiaoan-shicao/
      ...
  scripts/
    build_registry.py        ← 构建脚本
  .github/workflows/
    update-registry.yml
```

---

## 七、CDN 部署

### 7.1 架构

所有 registry 的静态文件统一部署到一个 `registry-deploy` 仓库，通过 Cloudflare Pages 托管。

域名：`registry.presto.app`

### 7.2 目录结构

```
registry-deploy/
  templates/
    registry.json
    gongwen/
      manifest.json
      README.md
      example.md
      preview-1.svg
      preview-2.svg
    jiaoan-shicao/
      ...
  plugins/                   ← 将来
    registry.json
    ...
  agent-skills/              ← 将来
    registry.json
    ...
```

### 7.3 URL 映射

```
https://registry.presto.app/templates/registry.json
https://registry.presto.app/templates/gongwen/manifest.json
https://registry.presto.app/templates/gongwen/preview-1.svg
```

### 7.4 数据流

```
template-registry CI → 生成文件 → push 到 registry-deploy/templates/
plugin-registry CI   → 生成文件 → push 到 registry-deploy/plugins/
                                         ↓
                              Cloudflare Pages 自动部署
                                         ↓
                              registry.presto.app 可访问
```

每个 registry 只写自己的子目录，互不冲突。

---

## 八、安装流程

### 8.1 从商店安装

```
前端 fetch registry.json → 显示商店列表
用户点安装 → POST /api/templates/{name}/install { owner, repo }
后端：
  1. GET github.com/repos/{owner}/{repo}/releases/latest
  2. 匹配当前平台的二进制资产（{os}_{arch}）
  3. 下载二进制（限 100MB）
  4. 验证 SHA256（从 SHA256SUMS）
  5. 运行 ./binary --manifest → 提取 manifest.json
  6. 写入 ~/.presto/templates/{name}/manifest.json
  7. 写入 ~/.presto/templates/{name}/presto-template-{name}
  8. 设置执行权限 0755
```

### 8.2 本地目录结构

```
~/.presto/
  templates/
    {name}/
      manifest.json                    ← 从二进制提取
      presto-template-{name}[.exe]     ← 可执行文件
  plugins/                             ← 将来
    {name}/
      ...
```

---

## 九、Presto 软件内商店 UI 规范

### 9.1 统一布局

所有页面左侧导航/列表宽度统一为 **180px**。

| 页面 | 左侧宽度 | 右侧 |
|------|---------|------|
| 设置 | 180px | 内容区（max-width: 600px，桌面端） |
| 批量转换 | 180px | 内容区（max-width: 640px，桌面端） |
| 模板商店 | 180px | 详情面板（flex: 1，桌面端有 max-width，网页端自适应） |
| 插件商店（将来） | 180px | 同上 |

### 9.2 商店页面路由

```
/store            → 模板商店
/plugins          → 插件商店（将来）
/agent-skills     → Agent Skills 商店（将来）
```

### 9.3 通用商店组件

三个商店共用 `StoreView.svelte` 组件，通过 props 区分：

```svelte
<StoreView
  type="template"
  registryUrl="https://registry.presto.app/templates/registry.json"
  installEndpoint="/api/templates"
/>
```

### 9.4 两种视图

**卡片网格视图**（未选中任何扩展时）：
- CSS Grid：`repeat(auto-fill, minmax(220px, 1fr))`
- 卡片内容：名称、信任标识、描述（2 行截断）、版本/作者

**Master-Detail 视图**（选中某个扩展时）：
- 左侧 180px：紧凑卡片列表，选中项高亮
- 右侧 flex: 1：详情面板（基本信息、实时预览 iframe、README、Schema、字体、安装按钮）
- 实时预览：iframe 嵌入 `/showcase/editor?registry={name}`

### 9.5 入口

设置页左侧导航中的"模板商店"tab → `goto('/store')`

---

## 十、模板开发者工作流

### 10.1 从 Starter 仓库创建

```
1. GitHub "Use this template" → 从 presto-template-starter-{lang} 创建仓库
2. git clone 到本地
3. 参考文件放入 reference/ 目录
4. 用 AI 编程工具（任意）开发
5. make preview → 在 Presto 中预览
6. git tag v1.0.0 && git push --tags → CI 自动构建发布
7. registry CI 自动收录
```

### 10.2 支持的语言

初始版本：Go、Rust、TypeScript (Bun)
后续可加：Python、Zig、Deno、Kotlin (GraalVM)、C#、Swift（仅 macOS/Linux）

### 10.3 开发者预览

使用 Presto 本身预览：

```bash
make preview
# 实际执行：
# 1. 编译当前平台的二进制
# 2. 复制到 ~/.presto/templates/{name}/
# 3. 运行 --manifest 提取 manifest.json
# 4. 打开/刷新 Presto
```

### 10.4 AI 辅助开发

每个 starter 仓库包含多 AI 工具的配置文件，指向统一的 CONVENTIONS.md：

| 文件 | 适配工具 |
|------|---------|
| `CONVENTIONS.md` | 通用内容（single source of truth） |
| `CLAUDE.md` | Claude Code |
| `AGENTS.md` | OpenAI Codex |
| `.cursor/rules` | Cursor |
| 其他 | 按需适配 |

---

## 十一、跨仓库协作

### 11.1 仓库清单

| 仓库 | 职责 | 状态 |
|------|------|------|
| `Presto-io/Presto` | 主应用（商店页面 + 安装 API） | 已有，需改造 |
| `Presto-io/Presto-Homepage` | 官网（模板商店展示页） | 已有，需小改 |
| `Presto-io/template-registry` | 模板注册表 CI | 待创建 |
| `Presto-io/plugin-registry` | 插件注册表 CI | 将来 |
| `Presto-io/agent-skill-registry` | Agent Skills 注册表 CI | 将来 |
| `Presto-io/registry-deploy` | CDN 部署仓库（Cloudflare Pages） | 待创建 |
| `Presto-io/presto-template-starter-go` | Go 模板脚手架 | 待创建 |
| `Presto-io/presto-template-starter-rust` | Rust 模板脚手架 | 待创建 |
| `Presto-io/presto-template-starter-typescript` | TypeScript 模板脚手架 | 待创建 |
| `Presto-io/create-presto-template` | 交互式 CLI 工具 | 待创建 |

### 11.2 共享 CI

组织级共享 workflow 仓库 `Presto-io/.github`，各 registry 引用：

```yaml
uses: Presto-io/.github/.github/workflows/registry-build.yml@main
with:
  type: template
  topic: presto-template
  deploy-path: templates/
```

### 11.3 协作规范

- 所有仓库的 CLAUDE.md 引用本文档（extension-spec.md）作为规范来源
- 跨仓库变更时，先更新本文档，再各仓库跟进
- 类型定义（TypeScript / Go struct）必须与本文档的 schema 完全一致
