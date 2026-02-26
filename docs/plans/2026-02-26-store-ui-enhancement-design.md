# 模板商店 UI 增强设计

> 日期: 2026-02-26
> 状态: 已批准
> 涉及仓库: Gopst (前端), registry-deploy (Stats API)

## 概述

为模板商店添加统计数据展示（star 数、下载量）、排序功能、导航优化、README 全高度展示、底部操作区增强，同时确保官网 iframe 嵌入兼容。

## 一、数据层：Stats API

### 架构

在 `registry-deploy`（Cloudflare Pages）添加 Functions + KV 存储。

```text
registry-deploy/
  functions/
    api/
      stats.ts              → GET /api/stats (批量获取)
      stats/
        [name]/
          download.ts       → POST /api/stats/:name/download
  templates/                → 现有静态文件不变
```

### 数据模型

KV 键值：

- `stats:{template-name}` → `{ stars: number, downloads: number }`
- `ratelimit:{ip-hash}:{template}` → TTL 24h（防刷）

### 数据来源

- **Stars**: GitHub Actions 定时任务（每小时）拉取各模板 GitHub 仓库 star 数，写入 KV
- **Downloads**: 桌面端安装成功后 POST `/api/stats/{name}/download`，KV 中 downloads++

### 限频策略

- Download 计数：同一 IP（CF-Connecting-IP hash）对同一模板 24h 内只计一次
- Stars 由 GitHub Actions 写入，不经过公开 API

### 前端类型扩展

```typescript
interface TemplateStats {
  stars?: number;
  downloads?: number;
}
```

前端启动时批量请求 `/api/stats`，合并到模板数据。数据不可用时不显示数字。

## 二、模板详情页头部

### 头部布局

```text
模板显示名称  [官方徽章]           [管理]  ⭐ 1.7k  ↓ 328
模板描述文字...
```

- Star 数和下载量：图标 + 数字，纯只读
- 数字格式：默认缩写（1.7k），点击 toggle 为精确数字（1734），再点回缩写（GitHub 风格）
- 管理按钮：颜色淡化（muted），放在统计数字左侧
  - 桌面端 + 已安装模板：显示，点击 `goto('/settings?panel=tpl-manage&focus={name}')`
  - Showcase 模式：隐藏
  - 未安装模板：隐藏

## 三、卡片网格排序

### 卡片网格布局

```text
🔍 搜索模板...          排序: [最新发布 ▾]

┌────┐ ┌────┐ ┌────┐ ┌────┐
│card│ │card│ │card│ │card│
│⭐12│ │⭐8 │ │⭐45│ │⭐3 │
└────┘ └────┘ └────┘ └────┘
```

### 排序选项

- 最新发布（默认，按发布日期降序）
- 最多星标（按 stars 降序）
- 最多下载（按 downloads 降序）

### 卡片统计

每张卡片底部小字显示 `⭐ 128  ↓ 1.2k`。

## 四、导航：面包屑 + 返回按钮

### 桌面端导航

| 视图     | 头部显示                     |
| -------- | ---------------------------- |
| 卡片网格 | `← 模板商店`                |
| 详情页   | `← 模板商店 › 公文模板`     |

- `←` 按钮始终回设置页（`goto(backRoute)`）
- 面包屑中「模板商店」可点击，回卡片网格（`selectedId = null`）

### Showcase 导航

| 视图     | 头部显示                 |
| -------- | ------------------------ |
| 卡片网格 | `模板商店`               |
| 详情页   | `模板商店 › 公文模板`    |

- 无 `←` 按钮
- 面包屑中「模板商店」可点击，回卡片网格

## 五、README 全高度展示

移除 `.readme-body` 的 `max-height: 400px; overflow-y: auto`，让 README 内容自然撑开。

## 六、底部操作区

### 操作区布局

```text
[安装] [管理]                              [↑ 回到顶部]
```

### 按钮逻辑

| 按钮     | 桌面端                                   | Showcase                                                                  |
| -------- | ---------------------------------------- | ------------------------------------------------------------------------- |
| 安装     | 正常安装逻辑                             | 「在 Presto 中打开」（`presto://install/{name}`，fallback 到下载页）      |
| 管理     | 已安装时显示，跳转设置页管理面板         | 隐藏                                                                      |
| 回到顶部 | 滚动超过阈值后显示，平滑滚动到顶部       | 隐藏（父页面控制滚动）                                                    |

## 七、iframe 兼容（Showcase 模式）

### 方案：自适应高度 iframe

iframe 内容不产生自身滚动，高度随内容自然撑开，滚动由父页面（官网）控制。

### 模式差异汇总

| 功能       | 桌面端 (mode=desktop)  | Showcase (mode=web)        |
| ---------- | ---------------------- | -------------------------- |
| Star 数    | 显示                   | 显示                       |
| 下载量     | 显示                   | 显示                       |
| 管理按钮   | 已安装时显示           | 隐藏                       |
| 安装按钮   | 正常安装               | `presto://` 唤起           |
| 排序下拉   | 可用                   | 可用                       |
| 返回按钮 ← | 显示（回设置页）       | 隐藏                       |
| 面包屑     | 显示                   | 显示                       |
| 回到顶部   | 显示                   | 隐藏                       |
| 内容滚动   | 组件内部滚动           | 自适应高度，父页面滚动     |

## 八、后续任务（不在本次范围）

### Presto URL Scheme 注册

需要在 Wails 桌面端注册 `presto://` 自定义 URL scheme，处理 `presto://install/{template-name}` 请求。详见独立提示词。

## 涉及文件

| 文件                                                        | 改动                                         |
| ----------------------------------------------------------- | -------------------------------------------- |
| `frontend/src/lib/components/StoreView.svelte`              | 主要改动：头部、排序、导航、README、底部按钮 |
| `frontend/src/lib/api/types.ts`                             | 新增 TemplateStats 类型                      |
| `frontend/src/lib/api/client.ts`                            | 新增 fetchStats()、reportDownload()          |
| `registry-deploy/functions/api/stats.ts`                    | 新建：批量获取统计                           |
| `registry-deploy/functions/api/stats/[name]/download.ts`    | 新建：下载计数                               |
| `.github/workflows/sync-stars.yml` (registry-deploy)        | 新建：定时同步 GitHub stars                  |
