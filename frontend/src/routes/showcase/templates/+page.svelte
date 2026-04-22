<script lang="ts">
  import { ArrowLeft, Search, Package, Trash2, Upload, Pencil } from 'lucide-svelte';
  import { mockTemplates } from '$lib/showcase/presets';

  // Build template data with additional fields to mimic real template management
  interface ShowcaseTemplate {
    name: string;
    displayName: string;
    description: string;
    version: string;
    author: string;
    keywords: string[];
  }

  const templates: ShowcaseTemplate[] = mockTemplates.map((t, i) => ({
    name: t.name.toLowerCase().replace(/[（）]/g, ''),
    displayName: t.name,
    description: i === 0 ? '符合 GB/T 9704-2012 标准的类公文排版'
      : i === 1 ? '将 Markdown 格式的实操教案转换为标准表格排版'
      : i === 2 ? '标准会议纪要格式，支持参会人员、议程、决议'
      : i === 3 ? '符合学术规范的论文排版，支持摘要、参考文献'
      : i === 4 ? '简洁美观的个人简历模板，支持多种布局'
      : i === 5 ? '标准合同协议格式，支持甲乙方信息、条款'
      : '标准周报格式，支持本周完成、下周计划、问题总结',
    version: i < 2 ? '1.0.0' : i < 4 ? '0.9.0' : '0.8.0',
    author: t.author,
    keywords: t.keywords,
  }));

  let tplSearch = $state('');
  let selectedKeywords: string[] = $state([]);

  let allKeywords = $derived(
    [...new Set(templates.flatMap(tpl => tpl.keywords))].sort()
  );

  let filteredTemplates = $derived(
    templates.filter(tpl => {
      const q = tplSearch.toLowerCase();
      const matchesSearch = !q ||
        tpl.displayName.toLowerCase().includes(q) ||
        tpl.description.toLowerCase().includes(q) ||
        tpl.keywords.some(k => k.toLowerCase().includes(q));
      const matchesKeywords = selectedKeywords.length === 0 ||
        selectedKeywords.every(k => tpl.keywords.includes(k));
      return matchesSearch && matchesKeywords;
    })
  );

  function toggleKeyword(kw: string) {
    if (selectedKeywords.includes(kw)) {
      selectedKeywords = selectedKeywords.filter(k => k !== kw);
    } else {
      selectedKeywords = [...selectedKeywords, kw];
    }
  }
</script>

<div class="page">
  <div class="page-header">
    <button class="btn-back" aria-label="返回编辑器">
      <ArrowLeft size={16} />
    </button>
    <h2>设置</h2>
  </div>

  <div class="settings-layout">
    <nav class="settings-nav">
      <button class="nav-item">通用</button>
      <button class="nav-item">帮助</button>
      <button class="nav-item">模板开发</button>
      <button class="nav-item">关于</button>
      <button class="nav-item">开源协议</button>
      <div class="nav-divider"></div>
      <button class="nav-item active">
        <Package size={14} />
        模板管理
      </button>
    </nav>

    <div class="settings-content-wrapper">
      <div class="panel-overlay">
        <div class="panel-header">
          <div class="panel-search">
            <Search size={14} />
            <input
              type="text"
              placeholder="搜索已安装模板…"
              bind:value={tplSearch}
            />
          </div>
          <button class="btn-import">
            <Upload size={14} />
            <span>从 ZIP 导入</span>
          </button>
        </div>

        {#if allKeywords.length > 0}
          <div class="keyword-filter-bar">
            {#each allKeywords as kw (kw)}
              <button
                class="keyword-chip"
                class:active={selectedKeywords.includes(kw)}
                onclick={() => toggleKeyword(kw)}
              >
                {kw}
              </button>
            {/each}
          </div>
        {/if}

        {#if filteredTemplates.length === 0}
          <div class="panel-empty">
            <Package size={32} />
            <p>{tplSearch ? '没有匹配的模板' : '暂无已安装模板'}</p>
          </div>
        {:else}
          <div class="tpl-list">
            {#each filteredTemplates as tpl (tpl.name)}
              <div class="tpl-row">
                <div class="tpl-info">
                  <div class="tpl-name-row">
                    <span class="tpl-name">{tpl.displayName}</span>
                    <span class="tpl-version">v{tpl.version}</span>
                  </div>
                  <p class="tpl-desc">{tpl.description}</p>
                  {#if tpl.keywords.length > 0}
                    <div class="tpl-keywords">
                      {#each tpl.keywords as kw (kw)}
                        <span class="keyword-badge">{kw}</span>
                      {/each}
                    </div>
                  {/if}
                  <span class="tpl-author">{tpl.author}</span>
                </div>
                <div class="tpl-actions">
                    <button class="btn-rename" aria-label="重命名">
                      <Pencil size={12} />
                    </button>
                    <button class="btn-uninstall" aria-label="卸载">
                      <Trash2 size={14} />
                      <span>卸载</span>
                    </button>
                  </div>
              </div>
            {/each}
          </div>
        {/if}
      </div>
    </div>
  </div>
</div>

<style>
  .page {
    padding: var(--space-xl);
    padding-top: 48px;
    height: 100%;
    display: flex;
    flex-direction: column;
  }
  .page-header {
    display: flex;
    align-items: center;
    gap: var(--space-md);
    margin-bottom: var(--space-xl);
    flex-shrink: 0;
  }
  h2 {
    margin: 0;
    font-size: 1.125rem;
    font-family: var(--font-ui);
    color: var(--color-text-bright);
  }
  .btn-back {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 28px;
    height: 28px;
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-sm);
    color: var(--color-text);
    cursor: pointer;
    transition: background var(--transition);
  }
  .btn-back:hover { background: var(--color-surface-hover); }
  .settings-layout {
    display: flex;
    gap: var(--space-xl);
    flex: 1;
    min-height: 0;
  }
  .settings-nav {
    width: 140px;
    flex-shrink: 0;
    display: flex;
    flex-direction: column;
    gap: 2px;
    position: sticky;
    top: 0;
    align-self: flex-start;
  }
  .nav-divider {
    height: 1px;
    background: var(--color-border);
    margin: var(--space-sm) var(--space-md);
  }
  .nav-item {
    display: flex;
    align-items: center;
    gap: var(--space-xs);
    text-align: left;
    padding: var(--space-sm) var(--space-md);
    background: none;
    border: none;
    border-radius: var(--radius-sm);
    color: var(--color-muted);
    font-size: 0.8125rem;
    font-weight: 500;
    cursor: pointer;
    transition: all var(--transition);
    white-space: nowrap;
  }
  .nav-item:hover { color: var(--color-text); background: var(--color-surface); }
  .nav-item.active { color: var(--color-accent); background: var(--color-surface); }
  .settings-content-wrapper {
    flex: 1;
    min-height: 0;
    max-width: 600px;
    position: relative;
  }
  .panel-overlay {
    position: absolute;
    inset: 0;
    background: var(--color-bg);
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }
  .panel-header {
    display: flex;
    gap: var(--space-sm);
    margin-bottom: var(--space-md);
    flex-shrink: 0;
  }
  .panel-header .panel-search {
    flex: 1;
    margin-bottom: 0;
  }
  .btn-import {
    display: inline-flex;
    align-items: center;
    gap: var(--space-xs);
    padding: var(--space-sm) var(--space-md);
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    color: var(--color-text);
    font-size: 0.75rem;
    cursor: pointer;
    transition: all var(--transition);
    white-space: nowrap;
    flex-shrink: 0;
  }
  .btn-import:hover { border-color: var(--color-accent); color: var(--color-accent); }
  .panel-search {
    display: flex;
    align-items: center;
    gap: var(--space-sm);
    padding: var(--space-sm) var(--space-md);
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    margin-bottom: var(--space-md);
    color: var(--color-muted);
    flex-shrink: 0;
  }
  .panel-search input {
    flex: 1;
    background: none;
    border: none;
    color: var(--color-text);
    font-size: 0.8125rem;
    font-family: var(--font-ui);
    outline: none;
  }
  .panel-search input::placeholder { color: var(--color-muted); }
  .panel-empty {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--space-sm);
    padding: var(--space-2xl);
    color: var(--color-muted);
  }
  .panel-empty p { margin: 0; }
  .keyword-filter-bar {
    display: flex;
    flex-wrap: wrap;
    gap: var(--space-xs);
    margin-bottom: var(--space-md);
    flex-shrink: 0;
  }
  .keyword-chip {
    padding: 2px 8px;
    border-radius: 10px;
    border: 1px solid var(--color-border);
    background: var(--color-surface);
    color: var(--color-muted);
    font-size: 0.6875rem;
    cursor: pointer;
    transition: all var(--transition);
  }
  .keyword-chip:hover { border-color: var(--color-accent); color: var(--color-text); }
  .keyword-chip.active {
    background: var(--color-accent);
    color: var(--color-bg);
    border-color: var(--color-accent);
  }
  .tpl-list {
    display: flex;
    flex-direction: column;
    gap: var(--space-sm);
    overflow-y: auto;
    flex: 1;
    scrollbar-gutter: stable;
  }
  .tpl-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-md);
    padding: var(--space-md) var(--space-lg);
    background: var(--color-surface);
    border-radius: var(--radius-md);
    border: 1px solid var(--color-border);
    transition: border-color var(--transition);
  }
  .tpl-row:hover { border-color: var(--color-surface-hover); }
  .tpl-info { flex: 1; min-width: 0; }
  .tpl-name-row {
    display: flex;
    align-items: baseline;
    gap: var(--space-sm);
    margin-bottom: 2px;
  }
  .tpl-name { font-size: 0.875rem; font-weight: 500; color: var(--color-text-bright); }
  .tpl-version { font-size: 0.75rem; color: var(--color-muted); font-family: var(--font-mono); }
  .tpl-desc {
    margin: 0 0 2px;
    font-size: 0.8125rem;
    color: var(--color-muted);
    line-height: 1.4;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .tpl-author { font-size: 0.75rem; color: var(--color-muted); }
  .tpl-keywords {
    display: flex;
    flex-wrap: wrap;
    gap: var(--space-xs);
    margin: 2px 0;
  }
  .keyword-badge {
    padding: 1px 6px;
    border-radius: 8px;
    background: var(--color-surface-hover);
    color: var(--color-muted);
    font-size: 0.625rem;
  }
  .tpl-actions {
    display: flex;
    align-items: center;
    gap: var(--space-xs);
    flex-shrink: 0;
  }
  .btn-rename {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 24px;
    height: 24px;
    background: none;
    border: 1px solid var(--color-border);
    border-radius: var(--radius-sm);
    color: var(--color-muted);
    cursor: pointer;
    transition: all var(--transition);
  }
  .btn-rename:hover { border-color: var(--color-accent); color: var(--color-accent); }
  .btn-uninstall {
    display: inline-flex;
    align-items: center;
    gap: var(--space-xs);
    padding: var(--space-xs) var(--space-sm);
    border-radius: var(--radius-sm);
    font-size: 0.75rem;
    font-weight: 500;
    cursor: pointer;
    transition: all var(--transition);
    white-space: nowrap;
    flex-shrink: 0;
    background: transparent;
    color: var(--color-danger);
    border: 1px solid var(--color-danger);
  }
  .btn-uninstall:hover { background: var(--color-danger); color: var(--color-on-danger); }
</style>
