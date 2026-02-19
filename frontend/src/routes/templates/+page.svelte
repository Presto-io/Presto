<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { listTemplates, discoverTemplates, installTemplate, deleteTemplate } from '$lib/api/client';
  import type { Template, GitHubRepo } from '$lib/api/types';
  import { ArrowLeft, Search, Package, Download, Trash2, Loader, ExternalLink } from 'lucide-svelte';

  let installed: Template[] = $state([]);
  let available: GitHubRepo[] = $state([]);
  let loading = $state(true);
  let installing = $state('');
  let activeTab: 'installed' | 'browse' = $state('installed');
  let searchQuery = $state('');
  let communityEnabled = $state(false);
  let browseLoaded = $state(false);

  let filteredInstalled = $derived(
    installed.filter(tpl => {
      const q = searchQuery.toLowerCase();
      return !q ||
        (tpl.displayName || tpl.name).toLowerCase().includes(q) ||
        tpl.description.toLowerCase().includes(q);
    })
  );

  let filteredAvailable = $derived(
    available.filter(repo => {
      const q = searchQuery.toLowerCase();
      return !q ||
        repo.name.toLowerCase().includes(q) ||
        (repo.description || '').toLowerCase().includes(q);
    })
  );

  onMount(async () => {
    communityEnabled = localStorage.getItem('communityTemplates') === 'true';
    try {
      installed = (await listTemplates()) ?? [];
    } catch {
      // silently handle
    } finally {
      loading = false;
    }
  });

  async function loadBrowse() {
    if (browseLoaded || !communityEnabled) return;
    loading = true;
    try {
      available = (await discoverTemplates()) ?? [];
      browseLoaded = true;
    } catch {
      // silently handle
    } finally {
      loading = false;
    }
  }

  function switchTab(tab: 'installed' | 'browse') {
    activeTab = tab;
    searchQuery = '';
    if (tab === 'browse') loadBrowse();
  }

  async function handleInstall(repo: GitHubRepo) {
    installing = repo.full_name;
    try {
      await installTemplate(repo.owner.login, repo.name);
      installed = await listTemplates();
      browseLoaded = false;
    } finally {
      installing = '';
    }
  }

  async function handleDelete(name: string) {
    if (!confirm(`确定卸载模板 "${name}"？`)) return;
    await deleteTemplate(name);
    installed = await listTemplates();
  }
</script>

<div class="page">
  <div class="page-header">
    <button class="btn-back" onclick={() => goto('/')} aria-label="返回编辑器">
      <ArrowLeft size={16} />
    </button>
    <h2>模板管理</h2>
  </div>

  <div class="tab-bar">
    <button
      class="tab"
      class:active={activeTab === 'installed'}
      onclick={() => switchTab('installed')}
    >
      已安装
      {#if installed.length > 0}
        <span class="tab-badge">{installed.length}</span>
      {/if}
    </button>
    {#if communityEnabled}
      <button
        class="tab"
        class:active={activeTab === 'browse'}
        onclick={() => switchTab('browse')}
      >
        浏览
      </button>
    {/if}
  </div>

  <div class="search-bar">
    <Search size={14} />
    <input
      type="text"
      placeholder={activeTab === 'installed' ? '搜索已安装模板…' : '搜索社区模板…'}
      bind:value={searchQuery}
    />
  </div>

  <div class="tab-content">
    {#if activeTab === 'installed'}
      {#if loading}
        <div class="empty">
          <Loader size={24} class="spin" />
          <p>加载中...</p>
        </div>
      {:else if filteredInstalled.length === 0}
        <div class="empty">
          <Package size={32} />
          <p>{searchQuery ? '没有匹配的模板' : '暂无已安装模板'}</p>
        </div>
      {:else}
        <div class="template-list">
          {#each filteredInstalled as tpl (tpl.name)}
            <div class="template-row">
              <div class="template-info">
                <div class="template-name-row">
                  <span class="template-name">{tpl.displayName || tpl.name}</span>
                  <span class="version">v{tpl.version}</span>
                </div>
                <p class="template-desc">{tpl.description}</p>
                <span class="template-author">{tpl.author}</span>
              </div>
              <button
                class="btn-uninstall"
                onclick={() => handleDelete(tpl.name)}
                aria-label="卸载 {tpl.name}"
              >
                <Trash2 size={14} />
                <span>卸载</span>
              </button>
            </div>
          {/each}
        </div>
      {/if}

    {:else if activeTab === 'browse'}
      {#if loading}
        <div class="empty">
          <Loader size={24} class="spin" />
          <p>加载中...</p>
        </div>
      {:else if filteredAvailable.length === 0}
        <div class="empty">
          <Package size={32} />
          <p>{searchQuery ? '没有匹配的模板' : '暂无可用模板'}</p>
        </div>
      {:else}
        <div class="template-list">
          {#each filteredAvailable as repo (repo.full_name)}
            <div class="template-row">
              <div class="template-info">
                <div class="template-name-row">
                  <span class="template-name">{repo.name}</span>
                  <a
                    href={repo.html_url}
                    target="_blank"
                    rel="noopener noreferrer"
                    class="repo-link"
                    aria-label="在 GitHub 上查看"
                  >
                    <ExternalLink size={12} />
                  </a>
                </div>
                <p class="template-desc">{repo.description}</p>
                <span class="template-author">{repo.owner.login}</span>
              </div>
              <button
                class="btn-install"
                onclick={() => handleInstall(repo)}
                disabled={installing === repo.full_name}
              >
                {#if installing === repo.full_name}
                  <Loader size={14} class="spin" />
                  <span>安装中...</span>
                {:else}
                  <Download size={14} />
                  <span>安装</span>
                {/if}
              </button>
            </div>
          {/each}
        </div>
      {/if}
    {/if}
  </div>
</div>

<style>
  .page {
    padding: var(--space-xl);
    padding-top: 48px;
    max-width: 700px;
    margin: 0 auto;
    overflow-y: auto;
    height: 100%;
  }
  .page-header {
    display: flex;
    align-items: center;
    gap: var(--space-md);
    margin-bottom: var(--space-lg);
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
  .tab-bar {
    display: flex;
    gap: var(--space-xs);
    margin-bottom: var(--space-md);
    border-bottom: 1px solid var(--color-border);
  }
  .tab {
    display: inline-flex;
    align-items: center;
    gap: var(--space-xs);
    padding: var(--space-sm) var(--space-md);
    background: none;
    border: none;
    border-bottom: 2px solid transparent;
    color: var(--color-muted);
    font-size: 0.8125rem;
    font-weight: 500;
    cursor: pointer;
    transition: all var(--transition);
  }
  .tab:hover { color: var(--color-text); }
  .tab.active { color: var(--color-accent); border-bottom-color: var(--color-accent); }
  .tab-badge {
    font-size: 0.6875rem;
    background: var(--color-surface-hover);
    padding: 1px 6px;
    border-radius: 10px;
    font-weight: 600;
  }
  .search-bar {
    display: flex;
    align-items: center;
    gap: var(--space-sm);
    padding: var(--space-sm) var(--space-md);
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    margin-bottom: var(--space-lg);
    color: var(--color-muted);
  }
  .search-bar input {
    flex: 1;
    background: none;
    border: none;
    color: var(--color-text);
    font-size: 0.8125rem;
    font-family: var(--font-ui);
    outline: none;
  }
  .search-bar input::placeholder { color: var(--color-muted); }
  .empty {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--space-sm);
    padding: var(--space-2xl);
    color: var(--color-muted);
  }
  .empty p { margin: 0; }
  .template-list {
    display: flex;
    flex-direction: column;
    gap: var(--space-sm);
  }
  .template-row {
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
  .template-row:hover { border-color: var(--color-surface-hover); }
  .template-info { flex: 1; min-width: 0; }
  .template-name-row {
    display: flex;
    align-items: baseline;
    gap: var(--space-sm);
    margin-bottom: 2px;
  }
  .template-name { font-size: 0.875rem; font-weight: 500; color: var(--color-text-bright); }
  .version { font-size: 0.75rem; color: var(--color-muted); font-family: var(--font-mono); }
  .template-desc {
    margin: 0 0 2px;
    font-size: 0.8125rem;
    color: var(--color-muted);
    line-height: 1.4;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .template-author { font-size: 0.75rem; color: var(--color-muted); }
  .repo-link {
    color: var(--color-muted);
    text-decoration: none;
    transition: color var(--transition);
  }
  .repo-link:hover { color: var(--color-accent); }
  .btn-uninstall, .btn-install {
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
  }
  .btn-uninstall {
    background: transparent;
    color: var(--color-danger);
    border: 1px solid var(--color-danger);
  }
  .btn-uninstall:hover {
    background: var(--color-danger);
    color: white;
  }
  .btn-install {
    background: var(--color-accent);
    color: var(--color-bg);
    border: none;
  }
  .btn-install:hover:not(:disabled) { opacity: 0.9; }
  .btn-install:disabled { opacity: 0.5; cursor: not-allowed; }
  :global(.spin) {
    animation: spin 1s linear infinite;
  }
  @keyframes spin {
    from { transform: rotate(0deg); }
    to { transform: rotate(360deg); }
  }
</style>
