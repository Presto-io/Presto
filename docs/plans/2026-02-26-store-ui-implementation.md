# 模板商店 UI 增强 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 为模板商店添加统计数据展示、排序、面包屑导航、README 全高度、底部操作区增强，确保 Showcase iframe 兼容。

**Architecture:** StoreView.svelte 是核心组件（1293 行），接受 mode='desktop'|'web' 控制行为差异。新增 Stats API 客户端从 registry.presto.app 获取统计数据，合并到模板列表。所有 UI 改动在 StoreView 内完成，通过 mode 条件控制桌面端/Showcase 差异。

**Tech Stack:** Svelte 5 (runes), TypeScript, lucide-svelte icons, DOMPurify, marked

**Design doc:** `docs/plans/2026-02-26-store-ui-enhancement-design.md`

---

## Task 1: 类型扩展与 Stats API 客户端

**Files:**
- Modify: `frontend/src/lib/api/types.ts:76-102`
- Modify: `frontend/src/lib/api/client.ts`

**Step 1: 在 types.ts 中添加 TemplateStats 接口**

在 `RegistryTemplate` 接口后添加：

```typescript
export interface TemplateStats {
  stars?: number;
  downloads?: number;
}

export type StatsMap = Record<string, TemplateStats>;
```

**Step 2: 在 client.ts 中添加 fetchStats 和 reportDownload**

在文件末尾添加：

```typescript
const STATS_BASE = 'https://registry.presto.app';

export async function fetchStats(): Promise<StatsMap> {
  try {
    const res = await fetch(`${STATS_BASE}/api/stats`);
    if (!res.ok) return {};
    return res.json();
  } catch {
    return {};
  }
}

export async function reportDownload(name: string): Promise<void> {
  try {
    await fetch(`${STATS_BASE}/api/stats/${encodeURIComponent(name)}/download`, {
      method: 'POST',
    });
  } catch {
    // silent - stats are best-effort
  }
}
```

**Step 3: Commit**

```bash
git add frontend/src/lib/api/types.ts frontend/src/lib/api/client.ts
git commit -m "feat: 添加模板统计类型和 Stats API 客户端"
```

---

## Task 2: StoreView Props 扩展与 Stats 数据加载

**Files:**
- Modify: `frontend/src/lib/components/StoreView.svelte:1-75`
- Modify: `frontend/src/routes/store-templates/+page.svelte`
- Modify: `frontend/src/routes/showcase/store-templates/+page@.svelte`

**Step 1: 扩展 StoreView Props**

在 `StoreView.svelte` 的 Props 接口中添加：

```typescript
interface Props {
  // ... existing props ...
  statsUrl?: string;           // Stats API base URL
  onInstallSuccess?: (name: string) => void;  // 安装成功回调（用于上报下载）
}
```

新增 props 解构：

```typescript
let {
  // ... existing ...
  statsUrl,
  onInstallSuccess,
}: Props = $props();
```

**Step 2: 添加 stats 状态和加载逻辑**

在 `StoreView.svelte` 的 UI state 区域添加：

```typescript
import type { StatsMap } from '$lib/api/types';

let statsMap = $state<StatsMap>({});

async function loadStats() {
  if (!statsUrl) return;
  try {
    const res = await fetch(statsUrl);
    if (res.ok) statsMap = await res.json();
  } catch { /* silent */ }
}
```

在 `onMount` 中调用 `loadStats()`：

```typescript
onMount(() => {
  loadRegistry();
  loadStats();
});
```

**Step 3: 添加数字格式化工具函数**

```typescript
function formatCount(n: number | undefined): string {
  if (n == null) return '';
  if (n >= 1000) return (n / 1000).toFixed(1).replace(/\.0$/, '') + 'k';
  return String(n);
}
```

**Step 4: 修改 handleInstall 添加下载上报**

在 `handleInstall` 的 try 块中，安装成功后调用回调：

```typescript
async function handleInstall(tpl: RegistryItem) {
  if (!installFn || installing || isInstalled(tpl.name)) return;
  installing = tpl.name;
  try {
    await installFn(tpl);
    onInstallSuccess?.(tpl.name);
  } catch (e) {
    console.error('Install failed:', e);
  } finally {
    installing = '';
  }
}
```

**Step 5: 更新路由页面传入新 props**

`store-templates/+page.svelte` 添加：

```svelte
<StoreView
  ...
  statsUrl="https://registry.presto.app/api/stats"
  onInstallSuccess={async (name) => {
    try {
      await fetch(`https://registry.presto.app/api/stats/${encodeURIComponent(name)}/download`, { method: 'POST' });
    } catch {}
  }}
/>
```

`showcase/store-templates/+page@.svelte` 添加：

```svelte
<StoreView
  ...
  statsUrl="https://registry.presto.app/api/stats"
/>
```

**Step 6: Commit**

```bash
git add frontend/src/lib/components/StoreView.svelte \
       frontend/src/routes/store-templates/+page.svelte \
       frontend/src/routes/showcase/store-templates/+page@.svelte
git commit -m "feat: StoreView 集成 Stats 数据加载与下载上报"
```

---

## Task 3: 面包屑导航改造

**Files:**
- Modify: `frontend/src/lib/components/StoreView.svelte:274-295` (page-header template)
- Modify: `frontend/src/lib/components/StoreView.svelte:382-385` (btn-back-grid)
- Modify: `frontend/src/lib/components/StoreView.svelte:569-605` (header CSS)
- Modify: `frontend/src/lib/components/StoreView.svelte:999-1013` (btn-back-grid CSS)

**Step 1: 替换 page-header 模板为面包屑**

将现有的 `.page-header` 区域（约 L278-295）替换为：

```svelte
<div class="page-header">
  {#if mode === 'desktop' && backRoute}
    <button class="btn-back" onclick={() => goto(backRoute!)} aria-label="返回设置">
      <ArrowLeft size={16} />
    </button>
  {/if}
  <nav class="breadcrumb">
    {#if selectedId && selectedTemplate}
      <button class="breadcrumb-link" onclick={() => selectedId = null}>{title}</button>
      <span class="breadcrumb-sep">›</span>
      <span class="breadcrumb-current">{selectedTemplate.displayName}</span>
    {:else}
      <h2>{title}</h2>
    {/if}
  </nav>
  {#if mode === 'desktop'}
    <button
      class="btn-refresh"
      onclick={() => refreshRegistry()}
      disabled={loading}
      aria-label="刷新"
    >
      <RefreshCw size={14} class={loading ? 'spin' : ''} />
    </button>
  {/if}
</div>
```

**Step 2: 删除详情页内的 btn-back-grid 按钮**

在 `.detail-header`（约 L382-394）中，删除 `btn-back-grid` 按钮，只保留 h3 和 trust badge：

```svelte
<div class="detail-header">
  <h3>{selectedTemplate.displayName}</h3>
  {#if selectedBadge}
    {@const BadgeIcon = selectedBadge.icon}
    <span class="trust-badge {selectedBadge.cls}" style={selectedBadge.color ? `color:${selectedBadge.color}` : ''}>
      <BadgeIcon size={14} />
      {selectedBadge.label}
    </span>
  {/if}
</div>
```

**Step 3: 添加面包屑 CSS**

```css
.breadcrumb {
  display: flex;
  align-items: center;
  gap: var(--space-xs);
  min-width: 0;
}
.breadcrumb h2 {
  margin: 0;
  font-size: 1.125rem;
  font-family: var(--font-ui);
  color: var(--color-text-bright);
}
.breadcrumb-link {
  background: none;
  border: none;
  padding: 0;
  font-size: 1.125rem;
  font-family: var(--font-ui);
  color: var(--color-accent);
  cursor: pointer;
  transition: opacity var(--transition);
  white-space: nowrap;
}
.breadcrumb-link:hover { opacity: 0.8; }
.breadcrumb-sep {
  color: var(--color-muted);
  font-size: 1rem;
  flex-shrink: 0;
}
.breadcrumb-current {
  font-size: 1.125rem;
  font-family: var(--font-ui);
  color: var(--color-text-bright);
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
```

**Step 4: 删除 btn-back-grid CSS**

删除 `.btn-back-grid` 相关的 CSS 规则（约 L999-1013），以及移动端响应式中的 `.btn-back-grid` 规则（约 L1277-1280）。

**Step 5: Commit**

```bash
git add frontend/src/lib/components/StoreView.svelte
git commit -m "ui: 面包屑导航替代冗余返回按钮"
```

---

## Task 4: 详情页头部添加统计数据和管理按钮

**Files:**
- Modify: `frontend/src/lib/components/StoreView.svelte` (detail-header template + CSS)

**Step 1: 导入新图标**

在 import 行添加 `Star, Settings`（或使用已有的 `Download`）：

```typescript
import { ..., Star, Settings } from 'lucide-svelte';
```

注意：检查 lucide-svelte 是否有 `Star` 图标，如果没有用 `StarIcon` 或其他替代。

**Step 2: 添加数字 toggle 状态**

```typescript
let showExactStats = $state(false);
```

**Step 3: 改造 detail-header 模板**

将 `.detail-header` 改为两行布局——第一行标题+徽章+右侧统计：

```svelte
<div class="detail-header">
  <div class="detail-title-row">
    <h3>{selectedTemplate.displayName}</h3>
    {#if selectedBadge}
      {@const BadgeIcon = selectedBadge.icon}
      <span class="trust-badge {selectedBadge.cls}" style={selectedBadge.color ? `color:${selectedBadge.color}` : ''}>
        <BadgeIcon size={14} />
        {selectedBadge.label}
      </span>
    {/if}
    <div class="detail-stats-actions">
      {#if mode === 'desktop' && isInstalled(selectedTemplate.name)}
        <button
          class="btn-manage"
          onclick={() => goto(`/settings?panel=tpl-manage&focus=${selectedTemplate.name}`)}
        >
          <Settings size={13} />
          <span>管理</span>
        </button>
      {/if}
      {#if statsMap[selectedTemplate.name]?.stars != null}
        <button class="stat-item" onclick={() => showExactStats = !showExactStats} title="Stars">
          <Star size={13} />
          <span>{showExactStats ? statsMap[selectedTemplate.name].stars : formatCount(statsMap[selectedTemplate.name].stars)}</span>
        </button>
      {/if}
      {#if statsMap[selectedTemplate.name]?.downloads != null}
        <button class="stat-item" onclick={() => showExactStats = !showExactStats} title="下载量">
          <Download size={13} />
          <span>{showExactStats ? statsMap[selectedTemplate.name].downloads : formatCount(statsMap[selectedTemplate.name].downloads)}</span>
        </button>
      {/if}
    </div>
  </div>
</div>
```

**Step 4: 添加 CSS**

```css
.detail-title-row {
  display: flex;
  align-items: center;
  gap: var(--space-md);
  flex-wrap: wrap;
}
.detail-stats-actions {
  display: flex;
  align-items: center;
  gap: var(--space-sm);
  margin-left: auto;
}
.stat-item {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  background: none;
  border: none;
  color: var(--color-muted);
  font-size: 0.8125rem;
  font-family: var(--font-mono);
  cursor: pointer;
  padding: 2px 6px;
  border-radius: var(--radius-sm);
  transition: color var(--transition), background var(--transition);
}
.stat-item:hover {
  color: var(--color-text);
  background: var(--color-surface);
}
.btn-manage {
  display: inline-flex;
  align-items: center;
  gap: var(--space-xs);
  padding: 4px 10px;
  background: none;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  color: var(--color-muted);
  font-size: 0.75rem;
  font-family: var(--font-ui);
  cursor: pointer;
  transition: all var(--transition);
}
.btn-manage:hover {
  color: var(--color-text);
  border-color: var(--color-accent-border);
}
```

**Step 5: Commit**

```bash
git add frontend/src/lib/components/StoreView.svelte
git commit -m "ui: 详情页头部添加统计数据和管理按钮"
```

---

## Task 5: 卡片网格添加统计数据和排序下拉

**Files:**
- Modify: `frontend/src/lib/components/StoreView.svelte` (filter-toolbar + card template + CSS)

**Step 1: 添加排序状态**

```typescript
type SortOption = 'latest' | 'stars' | 'downloads';
let sortBy = $state<SortOption>('latest');
```

**Step 2: 修改 filteredTemplates 加入排序逻辑**

在现有的 `filteredTemplates` derived 之后，添加排序：

```typescript
let sortedTemplates = $derived.by(() => {
  const list = [...filteredTemplates];
  switch (sortBy) {
    case 'stars':
      return list.sort((a, b) => (statsMap[b.name]?.stars ?? 0) - (statsMap[a.name]?.stars ?? 0));
    case 'downloads':
      return list.sort((a, b) => (statsMap[b.name]?.downloads ?? 0) - (statsMap[a.name]?.downloads ?? 0));
    case 'latest':
    default:
      return list.sort((a, b) => {
        const da = a.publishedAt ? new Date(a.publishedAt).getTime() : 0;
        const db = b.publishedAt ? new Date(b.publishedAt).getTime() : 0;
        return db - da;
      });
  }
});
```

然后将所有引用 `filteredTemplates` 的地方（分页、卡片渲染、侧边栏导航）改为引用 `sortedTemplates`：
- `totalPages` 中的 `filteredTemplates.length` → `sortedTemplates.length`
- `pagedTemplates` 中的 `filteredTemplates.slice(...)` → `sortedTemplates.slice(...)`
- 侧边栏 `{#each filteredTemplates as tpl}` → `{#each sortedTemplates as tpl}`
- 分页信息 `{filteredTemplates.length} 项` → `{sortedTemplates.length} 项`

**Step 3: 在搜索栏旁添加排序下拉**

在 `.filter-toolbar` 的 `.search-box` 后面添加排序下拉：

```svelte
<div class="search-sort-row">
  <div class="search-box">
    <!-- existing search content -->
  </div>
  <select class="sort-select" bind:value={sortBy}>
    <option value="latest">最新发布</option>
    <option value="stars">最多星标</option>
    <option value="downloads">最多下载</option>
  </select>
</div>
```

需要将 `.search-box` 和排序下拉包裹在一个 flex row 中。

**Step 4: 卡片底部添加统计数字**

在 `.card-footer` 中添加统计：

```svelte
<div class="card-footer">
  <span class="card-version">v{tpl.version}</span>
  <span class="card-author">{tpl.author}</span>
  {#if statsMap[tpl.name]?.stars != null || statsMap[tpl.name]?.downloads != null}
    <span class="card-stats">
      {#if statsMap[tpl.name]?.stars != null}
        <span class="card-stat"><Star size={10} /> {formatCount(statsMap[tpl.name].stars)}</span>
      {/if}
      {#if statsMap[tpl.name]?.downloads != null}
        <span class="card-stat"><Download size={10} /> {formatCount(statsMap[tpl.name].downloads)}</span>
      {/if}
    </span>
  {/if}
</div>
```

**Step 5: 添加 CSS**

```css
.search-sort-row {
  display: flex;
  align-items: center;
  gap: var(--space-md);
}
.search-sort-row .search-box {
  flex: 1;
}
.sort-select {
  flex-shrink: 0;
  padding: 7px var(--space-md);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  color: var(--color-text);
  font-family: var(--font-ui);
  font-size: 13px;
  cursor: pointer;
  transition: border-color var(--transition);
  outline: none;
  appearance: none;
  -webkit-appearance: none;
  background-image: url("data:image/svg+xml,...chevron-down...");
  background-repeat: no-repeat;
  background-position: right 8px center;
  padding-right: 28px;
}
.sort-select:focus {
  border-color: var(--color-accent-border);
}
.card-stats {
  display: flex;
  align-items: center;
  gap: var(--space-sm);
  margin-left: auto;
}
.card-stat {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  font-size: 0.6875rem;
  color: var(--color-muted);
}
```

**Step 6: Commit**

```bash
git add frontend/src/lib/components/StoreView.svelte
git commit -m "feat: 卡片网格添加统计数据展示和排序下拉"
```

---

## Task 6: README 全高度展示

**Files:**
- Modify: `frontend/src/lib/components/StoreView.svelte:1109-1110` (CSS)

**Step 1: 移除 readme-body 高度限制**

将 `.readme-body` CSS 中的：

```css
overflow-y: auto;
max-height: 400px;
```

删除这两行（或替换为无限制）。

**Step 2: Commit**

```bash
git add frontend/src/lib/components/StoreView.svelte
git commit -m "ui: README 移除高度限制，全高度展示"
```

---

## Task 7: 底部操作区增强

**Files:**
- Modify: `frontend/src/lib/components/StoreView.svelte` (detail-install template + CSS)

**Step 1: 添加滚动状态检测**

```typescript
let detailEl = $state<HTMLElement | null>(null);
let showScrollTop = $state(false);

function onDetailScroll() {
  if (!detailEl) return;
  showScrollTop = detailEl.scrollTop > 300;
}

function scrollToTop() {
  detailEl?.scrollTo({ top: 0, behavior: 'smooth' });
}
```

在 `.store-detail` 元素上绑定：

```svelte
<div class="store-detail" bind:this={detailEl} onscroll={onDetailScroll}>
```

**Step 2: 改造底部操作区模板**

将现有的 `detail-install` 区域替换为：

```svelte
<div class="detail-actions">
  <div class="actions-left">
    {#if mode === 'desktop' && installFn}
      {#if isInstalled(selectedTemplate.name)}
        <button class="btn-installed" disabled>
          <Check size={14} /><span>已安装</span>
        </button>
      {:else if installing === selectedTemplate.name}
        <button class="btn-installing" disabled>
          <Loader size={14} class="spin" /><span>安装中…</span>
        </button>
      {:else}
        <button class="btn-install" onclick={() => handleInstall(selectedTemplate!)}>
          <Download size={14} /><span>安装</span>
        </button>
      {/if}
    {:else if mode === 'web'}
      <a class="btn-install" href="presto://install/{selectedTemplate.name}" target="_blank" rel="noopener">
        <Download size={14} /><span>在 Presto 中打开</span>
      </a>
    {/if}
    {#if mode === 'desktop' && isInstalled(selectedTemplate.name)}
      <button
        class="btn-manage"
        onclick={() => goto(`/settings?panel=tpl-manage&focus=${selectedTemplate.name}`)}
      >
        <Settings size={13} /><span>管理</span>
      </button>
    {/if}
  </div>
  {#if mode === 'desktop' && showScrollTop}
    <button class="btn-scroll-top" onclick={scrollToTop} aria-label="回到顶部">
      <ArrowLeft size={14} style="transform:rotate(90deg)" />
      <span>回到顶部</span>
    </button>
  {/if}
</div>
```

**Step 3: 添加 CSS**

```css
.detail-actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: var(--space-xl);
  gap: var(--space-md);
}
.actions-left {
  display: flex;
  align-items: center;
  gap: var(--space-sm);
}
.btn-scroll-top {
  display: inline-flex;
  align-items: center;
  gap: var(--space-xs);
  padding: var(--space-sm) var(--space-md);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  color: var(--color-muted);
  font-size: 0.8125rem;
  cursor: pointer;
  transition: all var(--transition);
}
.btn-scroll-top:hover {
  color: var(--color-text);
  border-color: var(--color-accent-border);
}
```

**Step 4: Commit**

```bash
git add frontend/src/lib/components/StoreView.svelte
git commit -m "ui: 底部操作区添加管理按钮和回到顶部"
```

---

## Task 8: Showcase iframe 自适应高度

**Files:**
- Modify: `frontend/src/lib/components/StoreView.svelte` (CSS)
- Modify: `frontend/src/routes/showcase/store-templates/+page@.svelte`

**Step 1: web-mode 下移除固定高度和内部滚动**

在 StoreView CSS 中，为 `.web-mode` 添加：

```css
.page.web-mode {
  padding-top: var(--space-xl);
  height: auto;          /* 覆盖 height: 100% */
  overflow: visible;     /* 不产生内部滚动 */
}
.page.web-mode .store-detail {
  overflow: visible;     /* 详情面板也不滚动 */
}
.page.web-mode .card-grid {
  overflow: visible;     /* 卡片网格也不滚动 */
}
```

**Step 2: Showcase 页面确保 overflow 正确**

`+page@.svelte` 已有 `:global(.app), :global(#main-content) { overflow: auto !important; }` — 确认这与自适应高度兼容。如果 iframe 父页面需要知道内容高度，可以通过 `postMessage` 通知，但这属于官网侧的改动，当前不在范围内。

**Step 3: Commit**

```bash
git add frontend/src/lib/components/StoreView.svelte \
       frontend/src/routes/showcase/store-templates/+page@.svelte
git commit -m "ui: Showcase 模式自适应高度，移除内部滚动"
```

---

## Task 9: 手动验证

**Step 1: 启动开发服务器**

```bash
cd frontend && npm run dev
```

**Step 2: 验证清单**

- [ ] 桌面端卡片网格：排序下拉可用，卡片显示统计数字
- [ ] 桌面端详情页：面包屑导航正确，头部显示统计和管理按钮
- [ ] 桌面端详情页：README 无高度限制，底部有安装+管理+回到顶部
- [ ] 桌面端：回到顶部按钮在顶部时隐藏，滚动后显示
- [ ] 桌面端：面包屑中「模板商店」可点击回卡片网格
- [ ] 桌面端：← 按钮回设置页
- [ ] Showcase 模式：无 ← 按钮，无管理按钮，无回到顶部
- [ ] Showcase 模式：面包屑导航正确
- [ ] Showcase 模式：安装按钮显示「在 Presto 中打开」
- [ ] Showcase 模式：页面不产生内部滚动（自适应高度）
- [ ] Stats 数据不可用时，统计数字不显示（优雅降级）

---

## 后续任务（不在本次范围，独立仓库）

### registry-deploy: Cloudflare Pages Functions + KV

在 `registry-deploy` 仓库中：

1. 创建 `functions/api/stats.ts` — GET 批量获取统计
2. 创建 `functions/api/stats/[name]/download.ts` — POST 下载计数
3. 在 Cloudflare Dashboard 创建 KV namespace 并绑定
4. 添加限频逻辑（IP hash + 24h TTL）

### registry-deploy: GitHub Actions 同步 Stars

1. 创建 `.github/workflows/sync-stars.yml`
2. 定时任务每小时拉取各模板 GitHub 仓库 star 数
3. 通过 Cloudflare API 写入 KV

### Presto URL Scheme

见独立提示词（已在设计讨论中提供）。
