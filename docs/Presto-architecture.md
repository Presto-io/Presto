# Presto 官网与模板生态架构设计文档

本文档是 Presto 官网改造和模板生态建设的完整架构设计，作为各仓库 AI 助手的共享上下文。

---

## 一、项目全景

### 涉及的仓库

| 仓库 | 技术栈 | 职责 |
|---|---|---|
| `Presto-io/Presto` | Go + SvelteKit 2 + Svelte 5 + Wails v2 | 主应用（桌面 + Web） |
| `Presto-io/Presto-Homepage` | Astro 5（纯静态） | 官网 + 模板商店 |
| `Presto-io/template-registry` | GitHub Actions + Python | 模板注册表（静态索引） |

### 核心目标

1. **官网用真实 UI 组件替代静态截图**：Presto UI 更新后官网自动同步，无需手动截图
2. **建设真实的模板商店**：类似 Obsidian 插件商店，支持搜索、筛选、预览、安装
3. **为未来的插件商店和 Agent Skills 预留架构空间**

---

## 二、Showcase 模块（Presto 仓库内建）

### 2.1 架构决策

在 Presto 前端仓库内新增 `/showcase/` 路由，用真实 Svelte 组件渲染"半交互式"界面。
官网通过 iframe 嵌入这些路由。

**选择此方案的原因**：
- 零跨仓同步成本（组件直接复用，改了 UI 自动跟着变）
- iframe 天然样式隔离
- 官网保持纯 Astro 静态站，不增加构建复杂度

### 2.2 交互规则

**禁止的交互**（所有业务按钮的点击行为）：
- 导出 PDF、打开文件、保存、新建等功能按钮
- 模板选择器下拉
- 设置页面导航点击跳转
- 键盘输入（CodeMirror 设为 readOnly）
- 右键菜单

**允许的交互**：
- 所有 CSS :hover 效果
- 分割线拖拽（编辑器与预览面板之间）
- 双向滚动同步
- 右上角 proximity-reveal 按钮显隐（点击无效）
- 批量转换页文件拖拽分组 + 多选（Cmd+Click, Shift+Click）
- 模板管理页关键词 chip 筛选
- 设置页面滚动浏览
- CodeMirror 文本选择（readOnly 下仍可选择复制）

### 2.3 路由结构

```
frontend/src/routes/showcase/
  +layout.svelte                ← PrestoShell：全局点击拦截、主题同步、尺寸适配
  editor-gongwen/+page.svelte   ← 编辑器 - 公文模板
  editor-jiaoan/+page.svelte    ← 编辑器 - 教案模板
  batch/+page.svelte            ← 批量转换
  templates/+page.svelte        ← 模板管理
  drop/+page.svelte             ← 拖入文件动画
  hero/+page.svelte             ← Hero 打字动画
```

### 2.4 PrestoShell（showcase/+layout.svelte）

所有 showcase 页面的公共 layout，职责：

1. **事件拦截**：capture 阶段拦截 click/mousedown/keydown/contextmenu，通过白名单放行允许的交互（分割线 `.divider`、chip `.keyword-chip`、批量文件 `.batch-file-row`、CodeMirror `.cm-content` `.cm-scroller`、预览滚动 `.preview-scroll`）
2. **光标样式**：全局 `cursor: default`，可交互区域设对应光标
3. **主题同步**：继承 `prefers-color-scheme`，自动跟随系统深浅色
4. **视口适配**：固定尺寸渲染（建议 1200×800），CSS `transform: scale()` 适配 iframe
5. **隐藏全局 UI**：showcase 模式下隐藏 toast、confirm dialog 等（通过 URL 路径判断）

### 2.5 各页面详细需求

#### editor-gongwen / editor-jiaoan

- 复用 `+page.svelte` 的 split pane 布局
- 左侧 CodeMirror：加载对应模板的 example.md，设为 readOnly
- 右侧 Preview：加载对应模板的预编译 SVG
- 分割线可拖拽 + 双向滚动同步
- 右上角 proximity-reveal 按钮正常显隐（点击无效）
- 工具栏显示模板名称（静态文本，非下拉）
- 状态点脉冲动画

#### batch

- 复用 `batch/+page.svelte` 布局
- 预置 mock 文件列表（6 个文件，分属不同模板分组，部分自动检测）
- 文件可在分组间拖拽 + 多选 + 拖拽手柄 hover
- 转换按钮可见但点击无效

#### templates

- 复用 `settings/+page.svelte` 模板管理面板布局
- 预置 mock 模板列表：2 个真实 + 5 个 mock（会议纪要、学术论文、个人简历、合同协议、周报）
- 关键词 chip 筛选正常工作
- 模板卡片 hover 效果正常
- 操作按钮可见但点击无效

#### drop

- 进入视口时自动播放动画：
  1. 显示编辑器界面
  2. 1 秒后模拟文件图标从右上角飞入中央
  3. 触发 drop overlay（半透明遮罩 + 虚线边框 + "释放以导入文件"）
  4. overlay 保持 3 秒后淡出
  5. 间隔 5 秒后循环

#### hero

- 左侧 CodeMirror（readOnly）：初始为空，自动逐字输入公文 markdown
- 打字速度：50-80ms/字（随机），标点后暂停 200-400ms，换行后 300-500ms
- 总时长 5-8 秒，内容为公文模板前 10-15 行
- 右侧 Preview：分帧切换预渲染 SVG（opacity transition 300ms）
  - Frame 0：空白页（模板底纹/页眉）
  - Frame 1：标题 + 文号
  - Frame 2：标题 + 文号 + 主送单位 + 正文第一段
  - Frame 3：完整渲染（已有 gongwen-page-1.svg）

### 2.6 Showcase 与模板商店的集成

Showcase 需要支持**动态数据加载**，供模板商店详情页使用：

```
/showcase/editor?registry=gongwen
```

Showcase 页面根据 URL 参数，从 registry CDN fetch 对应模板的 `example.md` + `preview-*.svg`，渲染编辑器 + 预览的 split pane 布局。

这使 Showcase 成为通用的"模板预览引擎"，新模板上架不需要改 Presto 代码。

### 2.7 Mock 数据

集中管理在 `frontend/src/lib/showcase/` 目录：

```
frontend/src/lib/showcase/
  presets.ts              ← 各页面的预置数据
  svg/                    ← 预编译 SVG（公文、教案）
  hero-frames/            ← Hero 打字动画分帧 SVG
```

### 2.8 构建配置

- Showcase 路由通过 `adapter-static` 构建为独立 HTML
- 桌面构建可通过环境变量排除（或不排除，不影响功能）
- 确保 `/showcase/*` 路径可直接 URL 访问

---

## 三、模板系统架构

### 3.1 模板产物

每个模板 Release 只有两个文件：

```
presto-template-{name}    ← Go 二进制（//go:embed 内嵌 template_head.typ + example.md + manifest.json）
manifest.json              ← 元数据
```

二进制 CLI 协议：

| Flag | 行为 |
|---|---|
| `--manifest` | 输出 manifest.json |
| `--example` | 输出 example.md |
| （无 flag） | stdin 接收 markdown，stdout 输出 Typst 源码 |

### 3.2 manifest.json

```jsonc
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

`frontmatterSchema` 用途：商店详情页展示支持的字段、未来 Agent Skills 的 AI 接口文档、未来客户端可视化表单生成。

### 3.3 分类体系

分类（category）和关键词（keywords）分开。分类是一级导航，keywords 是二级筛选 chip。

```typescript
type TemplateCategory =
  | 'government'   // 政务
  | 'education'    // 教育（教案、考勤表、成绩册、比赛表等）
  | 'business'     // 商务/办公
  | 'academic'     // 学术
  | 'legal'        // 法务
  | 'resume'       // 简历/求职
  | 'creative'     // 创意/设计
  | 'other'        // 其他
```

**空分类不显示**——前端渲染时过滤掉没有模板的分类。

### 3.4 分发模型（解耦架构）

**核心原则**：发行版不包含任何模板可执行文件，模板与应用更新完全解耦。

**获取方式**：

| 方式 | 场景 | 验证 |
|------|------|------|
| 模板商店在线安装 | 联网环境 | SHA256 自动验证（后台静默） |
| ZIP 导入 | 离线环境 / 批量部署 | SHA256 对比 Registry 缓存 |
| URL 手动安装 | 开发者测试 | 无验证，标记为"未收录" |

**ZIP 导入验证流程**：

1. 解压 ZIP，读取二进制文件
2. 计算二进制的 `sha256.Sum256(binData)`
3. 从本地 Registry 缓存查找该模板 + 当前平台的期望 SHA256
4. 验证结果（四种状态）：

| 状态 | 含义 | 处理 |
|------|------|------|
| `verified` | SHA256 匹配 | 安装，绿色提示"已验证" |
| `not_in_registry` | 注册表中无此模板 | 安装，黄色提示"无法验证来源" |
| `pending` | 注册表缓存不可用（离线且无缓存） | 安装，蓝色提示"待验证，联网后可确认" |
| `mismatch` | SHA256 不匹配 | **拒绝安装**（可能被篡改） |

**Registry 缓存**（`~/.presto/registry-cache.json`）：

- 启动时异步刷新，不阻塞启动
- 缓存有效期 1 小时，过期后自动刷新
- CDN 不可达时使用本地缓存
- 缓存也不存在时，验证结果标记为 `pending`

**首次运行**：应用启动时无任何模板，模板选择器引导用户前往模板商店安装。

### 3.5 安全与信任

信任分级：

| 级别 | 标识 | 条件 |
|---|---|---|
| 官方 | 蓝色盾牌 | Presto-io 组织发布 |
| 已验证 | 绿色对勾 | 通过自动化审核 + 签名验证 |
| 社区 | 无标识 | 仅收录，未审核 |
| 未收录 | 警告标识 | 用户手动输入 URL 安装 |

- 软件内安装：后台静默验证 SHA256（防 MITM），用户无感知
- 手动安装：一律视为未验证
- 签名方案先用 GitHub 身份 + SHA256 起步（方案 A），后期可选 cosign（方案 B）

### 3.6 字体处理

- `requiredFonts` 中 `url` 为字体信息页（人工访问），`downloadUrl` 为直链（开源字体才有）
- 鼓励模板开发者尽量使用开源字体
- 开源字体可自动下载，商业字体引导用户去官网
- 浏览器端字体检测：Local Font Access API（Chrome/Edge），用户手动点击"检测本地字体"按钮触发，不自动请求权限

---

## 四、静态注册表（template-registry）

### 4.1 设计思路

类似 Homebrew tap 模式。建立 `Presto-io/template-registry` 仓库，定时 Action 构建静态索引。商店页面只 fetch 静态 JSON，零 API 调用，无限流量。

### 4.2 仓库结构

```
Presto-io/template-registry/
  registry.json                         ← 精简索引（商店首页卡片列表用）
  templates/
    gongwen/
      manifest.json                     ← 从 Release 复制
      README.md                         ← 从模板仓库首页获取
      example.md                        ← ./binary --example 的输出
      preview-1.svg                     ← Action 自动编译生成
      preview-2.svg
    jiaoan-shicao/
      manifest.json
      README.md
      example.md
      preview-1.svg
  scripts/
    build_registry.py                   ← 构建脚本
    download_fonts.py                   ← 字体下载脚本
  Dockerfile                            ← 沙箱镜像（含 Typst CLI + 基础字体）
  .github/workflows/
    update-registry.yml
```

### 4.3 registry.json 结构

```jsonc
{
  "version": 1,
  "updatedAt": "2026-02-21T07:00:00Z",
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

### 4.4 SVG 生成管线

```
下载 Release 的模板二进制
  → ./binary --example → example.md
  → cat example.md | ./binary → output.typ
  → typst compile --font-path ./fonts/ output.typ → preview-{n}.svg
```

三步串联。example.md 也保存到 registry（供 showcase 编辑器左侧显示）。

### 4.5 安全：运行社区二进制

将"运行不可信二进制"和"有写权限"拆成两个 GitHub Actions job：

- **Job 1**（`permissions: contents: read`，不传 secrets）：运行模板二进制，生成 Typst 源码和 example.md，通过 artifacts 传递
- **Job 2**（`permissions: contents: write`）：运行 Typst CLI（可信），编译 SVG，commit + push

即使恶意二进制在 Job 1 搞事，它没有 token，无法改仓库。

### 4.6 增量更新与手动触发

```yaml
on:
  schedule:
    - cron: '0 */6 * * *'        # 每 6 小时
  workflow_dispatch:               # 手动触发
    inputs:
      force_rebuild:
        description: '强制重建所有模板'
        type: boolean
        default: false
```

增量检测：对比每个模板仓库最新 Release tag 与 registry 记录的版本，只处理有更新的。同时检测新仓库（搜索 `topic:presto-template`）。

### 4.7 Hero 分帧 SVG

同样的管线，截取 example.md 不同长度版本（前 3 行、前 8 行、前 15 行、完整），分别编译为 SVG，存为 `hero-frame-0.svg` ~ `hero-frame-3.svg`。

---

## 五、模板商店页面（Homepage）

### 5.1 位置与技术

Homepage 仓库（Astro）新增 `/templates` 路由。使用 Svelte island 组件实现交互：

```astro
---
import TemplateStore from '../components/TemplateStore.svelte';
---
<Layout>
  <TemplateStore client:load registryUrl="https://..." />
</Layout>
```

需要添加 `@astrojs/svelte` 依赖。只有商店页面用 Svelte，其他页面保持纯静态。

### 5.2 UI 设计：Obsidian 式 master-detail

```
┌─────────────────────────────────────────────────┐
│  🔍 搜索...        [教育] [政务] [简历] ...      │
├──────────────┬──────────────────────────────────┤
│  ┌────────┐  │   公文模板              🔵 官方    │
│  │ 公文 ✦ │  │   符合 GB/T 9704-2012 标准...     │
│  └────────┘  │   [政务] [公文] [国标]             │
│  ┌────────┐  │   v1.2.0 · MIT · mrered           │
│  │ 教案   │  │                                   │
│  └────────┘  │   ┌──────────────────────────┐   │
│  ┌────────┐  │   │  Presto Showcase iframe  │   │
│  │ 会议纪要│  │   │  编辑器 │ 预览            │   │
│  └────────┘  │   │  ← 可拖拽 →              │   │
│  ...         │   └──────────────────────────┘   │
│              │   ## 说明（README 渲染）           │
│              │   所需字体 [检测本地字体]          │
│              │   仓库 · 兼容版本 · SHA256         │
└──────────────┴──────────────────────────────────┘
```

- 左侧：可滚动卡片列表 + 搜索框 + 分类/关键词 chips
- 右侧：选中模板的详情页
- 点击卡片 → 右侧显示详情，左侧高亮选中
- 返回时保持左侧列表滚动位置
- URL 用 `history.replaceState` 同步（`/templates?id=gongwen`），支持直链分享

### 5.3 详情页内容

- 模板名称、作者、版本、分类 chips、信任标识
- Live Preview：iframe 嵌入 `/showcase/editor?registry={name}`
  - 可拖拽分割线、可滚动、可选中文本、不可编辑
  - 数据来自 registry（example.md + preview SVG）
- README 渲染（markdown → HTML）
- frontmatterSchema 展示（支持的元数据字段）
- 所需字体列表 + "检测本地字体"按钮（Local Font Access API，Chrome/Edge，用户手动触发）
- 仓库链接、兼容版本、SHA256

### 5.4 数据获取

Astro 构建时 fetch `registry.json`，生成纯静态 HTML。registry 更新后触发 Homepage 重新构建。

---

## 六、首页导航更新

```
[Features] [Showcase] [Templates] [Plugins (coming soon)] [Agent Skills (coming soon)] [Download]
```

- Templates：链接到 `/templates` 商店页面
- Plugins / Agent Skills：灰色不可点击按钮，hover 显示 tooltip "即将推出"

---

## 七、跨仓库协作模式

各仓库分开管理，用本文档作为共享上下文：

```
用户 ──→ Presto AI（附带本文档）──→ 改 Presto 仓库
用户 ──→ Homepage AI（附带本文档）──→ 改 Homepage 仓库
用户 ──→ Registry AI（附带本文档）──→ 改 Registry 仓库
```

---

## 八、待后续讨论的话题

详见 `future-topics.md`：
1. 模板初始化脚手架仓库
2. 模板签名验证详细设计
3. 字体缺失检测的客户端实现
4. Agent Skills 具体架构
5. 插件系统设计
