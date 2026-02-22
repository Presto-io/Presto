<script lang="ts">
  import { onMount } from 'svelte';
  import { ArrowLeft, Search, Loader, ExternalLink, Download, Check, RefreshCw, ShieldCheck, Shield, Users } from 'lucide-svelte';
  import { goto } from '$app/navigation';
  import { registryStore } from '$lib/stores/registry.svelte';
  import { templateStore } from '$lib/stores/templates.svelte';
  import { installTemplate } from '$lib/api/client';
  import type { RegistryTemplate } from '$lib/api/types';

  let searchQuery = $state('');
  let activeCategory = $state<string | null>(null);
  let selectedId = $state<string | null>(null);
  let installing = $state('');
  let readmeContent = $state('');
  let readmeLoading = $state(false);

  let registry = $derived(registryStore.registry);
  let loading = $derived(registryStore.loading);
  let error = $derived(registryStore.error);

  const trustBadge = {
    official: { label: '官方', cls: 'trust-official', color: '#3b82f6', icon: ShieldCheck },
    verified: { label: '已验证', cls: 'trust-verified', color: '#22c55e', icon: Shield },
    community: { label: '社区', cls: 'trust-community', color: '', icon: Users },
  } as const;

  let filteredTemplates = $derived.by(() => {
    if (!registry) return [];
    return registry.templates.filter(tpl => {
      const q = searchQuery.toLowerCase();
      const matchesSearch = !q ||
        tpl.displayName.toLowerCase().includes(q) ||
        tpl.name.toLowerCase().includes(q) ||
        tpl.description.toLowerCase().includes(q) ||
        tpl.keywords.some(k => k.toLowerCase().includes(q));
      const matchesCategory = !activeCategory || tpl.category === activeCategory;
      return matchesSearch && matchesCategory;
    });
  });

  let selectedTemplate = $derived(
    registry?.templates.find(t => t.name === selectedId) ?? null
  );

  let installedNames = $derived(
    new Set(templateStore.templates.map(t => t.name))
  );

  const isDesktop = typeof window !== 'undefined' && !!(window as any).go;

  function isInstalled(name: string): boolean {
    return installedNames.has(name);
  }

  async function handleInstall(tpl: RegistryTemplate) {
    if (installing || isInstalled(tpl.name)) return;
    installing = tpl.name;
    try {
      const url = new URL(tpl.repository);
      const parts = url.pathname.slice(1).split('/');
      const owner = parts[0];
      const repo = parts[1];
      await installTemplate(owner, repo);
      await templateStore.refresh();
    } catch (e) {
      console.error('Install failed:', e);
    } finally {
      installing = '';
    }
  }

  async function loadReadme(name: string) {
    readmeLoading = true;
    readmeContent = '';
    try {
      const res = await fetch(`https://registry.presto.app/templates/${name}/README.md`);
      if (res.ok) {
        readmeContent = await res.text();
      }
    } catch {
      // silent
    } finally {
      readmeLoading = false;
    }
  }

  function selectTemplate(id: string) {
    selectedId = id;
    loadReadme(id);
  }

  function openExternal(url: string) {
    if ((window as any).runtime?.BrowserOpenURL) {
      (window as any).runtime.BrowserOpenURL(url);
    } else {
      window.open(url, '_blank', 'noopener,noreferrer');
    }
  }

  onMount(() => {
    registryStore.load();
    templateStore.load();
  });
</script>

<div class="page">
  <div class="page-header">
    <button class="btn-back" onclick={() => goto('/settings')} aria-label="返回设置">
      <ArrowLeft size={16} />
    </button>
    <h2>模板商店</h2>
    <button
      class="btn-refresh"
      onclick={() => registryStore.refresh()}
      disabled={loading}
      aria-label="刷新"
    >
      <RefreshCw size={14} class={loading ? 'spin' : ''} />
    </button>
  </div>

  {#if loading && !registry}
    <div class="store-empty">
      <Loader size={24} class="spin" />
      <p>加载模板列表…</p>
    </div>
  {:else if error && !registry}
    <div class="store-empty">
      <p class="error-text">加载失败：{error}</p>
      <button class="btn-retry" onclick={() => registryStore.refresh()}>重试</button>
    </div>
  {:else if registry}
    <!-- Search + Category Chips -->
    <div class="store-toolbar">
      <div class="store-search">
        <Search size={14} />
        <input
          type="text"
          placeholder="搜索模板…"
          bind:value={searchQuery}
        />
      </div>
      <div class="category-chips">
        <button
          class="chip"
          class:active={!activeCategory}
          onclick={() => activeCategory = null}
        >全部</button>
        {#each registry.categories as cat (cat.id)}
          <button
            class="chip"
            class:active={activeCategory === cat.id}
            onclick={() => activeCategory = activeCategory === cat.id ? null : cat.id}
          >{cat.label.zh}</button>
        {/each}
      </div>
    </div>

    {#if selectedId && selectedTemplate}
      <!-- Master-Detail View -->
      <div class="master-detail">
        <nav class="store-nav">
          {#each filteredTemplates as tpl (tpl.name)}
            {@const badge = trustBadge[tpl.trust]}
            <button
              class="nav-tpl-item"
              class:active={selectedId === tpl.name}
              onclick={() => selectTemplate(tpl.name)}
            >
              <span class="nav-tpl-name">{tpl.displayName}</span>
              <span class="nav-trust-dot" style="background:{badge.color}"></span>
            </button>
          {/each}
        </nav>

        <div class="store-detail" class:desktop={isDesktop}>
          <!-- Header -->
          <div class="detail-header">
            <h3>{selectedTemplate.displayName}</h3>
            {@const badge = trustBadge[selectedTemplate.trust]}
            {@const BadgeIcon = badge.icon}
            <span class="trust-badge {badge.cls}" style={badge.color ? `color:${badge.color}` : ''}>
              <BadgeIcon size={14} />
              {badge.label}
            </span>
          </div>

          <!-- Description -->
          <p class="detail-desc">{selectedTemplate.description}</p>

          <!-- Keywords -->
          {#if selectedTemplate.keywords.length > 0}
            <div class="detail-keywords">
              {#each selectedTemplate.keywords as kw (kw)}
                <span class="keyword-chip">{kw}</span>
              {/each}
            </div>
          {/if}

          <!-- Meta -->
          <div class="detail-meta">
            <span>v{selectedTemplate.version}</span>
            <span class="meta-sep">·</span>
            <span>{selectedTemplate.author}</span>
            <span class="meta-sep">·</span>
            <span>{selectedTemplate.license}</span>
          </div>

          <!-- Preview iframe -->
          <div class="detail-preview">
            <iframe
              src="/showcase/editor?registry={selectedTemplate.name}"
              sandbox="allow-scripts allow-same-origin"
              loading="lazy"
              title="模板预览"
            ></iframe>
          </div>

          <!-- README -->
          {#if readmeLoading}
            <div class="readme-loading">
              <Loader size={16} class="spin" />
              <span>加载 README…</span>
            </div>
          {:else if readmeContent}
            <div class="detail-readme">
              <h4>README</h4>
              <pre class="readme-text">{readmeContent}</pre>
            </div>
          {/if}

          <!-- Repository -->
          <div class="detail-repo">
            <a
              href={selectedTemplate.repository}
              onclick={(e) => { e.preventDefault(); openExternal(selectedTemplate!.repository); }}
              class="repo-link"
            >
              查看源码
              <ExternalLink size={12} />
            </a>
          </div>

          <!-- Install button -->
          <div class="detail-install">
            {#if isInstalled(selectedTemplate.name)}
              <button class="btn-installed" disabled>
                <Check size={14} />
                <span>已安装</span>
              </button>
            {:else if installing === selectedTemplate.name}
              <button class="btn-installing" disabled>
                <Loader size={14} class="spin" />
                <span>安装中…</span>
              </button>
            {:else}
              <button class="btn-install" onclick={() => handleInstall(selectedTemplate!)}>
                <Download size={14} />
                <span>安装</span>
              </button>
            {/if}
          </div>
        </div>
      </div>
    {:else}
      <!-- Card Grid View -->
      {#if filteredTemplates.length === 0}
        <div class="store-empty">
          <p>{searchQuery ? '没有匹配的模板' : '暂无可用模板'}</p>
        </div>
      {:else}
        <div class="card-grid">
          {#each filteredTemplates as tpl (tpl.name)}
            {@const badge = trustBadge[tpl.trust]}
            {@const BadgeIcon = badge.icon}
            <button class="tpl-card" onclick={() => selectTemplate(tpl.name)}>
              <div class="card-header">
                <span class="card-name">{tpl.displayName}</span>
                <span class="card-trust {badge.cls}" style={badge.color ? `color:${badge.color}` : ''}>
                  <BadgeIcon size={12} />
                  {badge.label}
                </span>
              </div>
              <p class="card-desc">{tpl.description}</p>
              <div class="card-footer">
                <span class="card-version">v{tpl.version}</span>
                <span class="card-author">{tpl.author}</span>
              </div>
            </button>
          {/each}
        </div>
      {/if}
    {/if}
  {/if}
</div>

<style>
  .page {
    padding: var(--space-xl);
    padding-top: 48px;
    height: 100%;
    display: flex;
    flex-direction: column;
  }
  h2 {
    margin: 0;
    font-size: 1.125rem;
    font-family: var(--font-ui);
    color: var(--color-text-bright);
  }
  .page-header {
    display: flex;
    align-items: center;
    gap: var(--space-md);
    margin-bottom: var(--space-xl);
    flex-shrink: 0;
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
  .btn-refresh {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 28px;
    height: 28px;
    background: none;
    border: 1px solid var(--color-border);
    border-radius: var(--radius-sm);
    color: var(--color-muted);
    cursor: pointer;
    transition: all var(--transition);
    margin-left: auto;
  }
  .btn-refresh:hover { color: var(--color-accent); border-color: var(--color-accent); }
  .btn-refresh:disabled { opacity: 0.5; cursor: not-allowed; }

  /* Toolbar */
  .store-toolbar {
    display: flex;
    flex-direction: column;
    gap: var(--space-md);
    margin-bottom: var(--space-xl);
    flex-shrink: 0;
  }
  .store-search {
    display: flex;
    align-items: center;
    gap: var(--space-sm);
    padding: var(--space-sm) var(--space-md);
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    color: var(--color-muted);
  }
  .store-search input {
    flex: 1;
    background: none;
    border: none;
    color: var(--color-text);
    font-size: 0.8125rem;
    font-family: var(--font-ui);
    outline: none;
  }
  .store-search input::placeholder { color: var(--color-muted); }
  .category-chips {
    display: flex;
    flex-wrap: wrap;
    gap: var(--space-xs);
  }
  .chip {
    padding: 2px 10px;
    border-radius: 10px;
    border: 1px solid var(--color-border);
    background: var(--color-surface);
    color: var(--color-muted);
    font-size: 0.75rem;
    cursor: pointer;
    transition: all var(--transition);
  }
  .chip:hover {
    border-color: var(--color-accent);
    color: var(--color-text);
  }
  .chip.active {
    background: var(--color-accent);
    color: var(--color-bg);
    border-color: var(--color-accent);
  }

  /* Empty / Loading */
  .store-empty {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--space-sm);
    padding: var(--space-2xl);
    color: var(--color-muted);
    flex: 1;
    justify-content: center;
  }
  .store-empty p { margin: 0; }
  .error-text { color: var(--color-danger); }
  .btn-retry {
    padding: var(--space-sm) var(--space-md);
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-sm);
    color: var(--color-text);
    font-size: 0.8125rem;
    cursor: pointer;
    transition: all var(--transition);
  }
  .btn-retry:hover { border-color: var(--color-accent); color: var(--color-accent); }

  /* Card Grid */
  .card-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
    gap: var(--space-md);
    overflow-y: auto;
    flex: 1;
  }
  .tpl-card {
    display: flex;
    flex-direction: column;
    padding: var(--space-lg);
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    cursor: pointer;
    transition: all var(--transition);
    text-align: left;
  }
  .tpl-card:hover {
    border-color: var(--color-accent-border);
    background: var(--color-surface-hover);
  }
  .card-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-sm);
    margin-bottom: var(--space-xs);
  }
  .card-name {
    font-size: 0.875rem;
    font-weight: 600;
    color: var(--color-text-bright);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .card-trust {
    display: inline-flex;
    align-items: center;
    gap: 2px;
    font-size: 0.625rem;
    font-weight: 500;
    white-space: nowrap;
    flex-shrink: 0;
  }
  .trust-official { color: #3b82f6; }
  .trust-verified { color: #22c55e; }
  .trust-community { color: var(--color-muted); }
  .card-desc {
    margin: 0;
    font-size: 0.8125rem;
    color: var(--color-muted);
    line-height: 1.4;
    overflow: hidden;
    display: -webkit-box;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
    flex: 1;
  }
  .card-footer {
    display: flex;
    justify-content: space-between;
    margin-top: var(--space-sm);
    font-size: 0.75rem;
    color: var(--color-muted);
  }
  .card-version { font-family: var(--font-mono); }

  /* Master-Detail */
  .master-detail {
    display: flex;
    gap: var(--space-xl);
    flex: 1;
    min-height: 0;
  }
  .store-nav {
    width: 180px;
    flex-shrink: 0;
    display: flex;
    flex-direction: column;
    gap: 2px;
    overflow-y: auto;
  }
  .nav-tpl-item {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-xs);
    text-align: left;
    padding: var(--space-sm) var(--space-md);
    background: none;
    border: none;
    border-radius: var(--radius-sm);
    color: var(--color-muted);
    font-size: 0.8125rem;
    cursor: pointer;
    transition: all var(--transition);
  }
  .nav-tpl-item:hover {
    color: var(--color-text);
    background: var(--color-surface);
  }
  .nav-tpl-item.active {
    color: var(--color-accent);
    background: var(--color-accent-bg);
  }
  .nav-tpl-name {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .nav-trust-dot {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    flex-shrink: 0;
  }

  /* Detail panel */
  .store-detail {
    flex: 1;
    overflow-y: auto;
    padding-right: var(--space-md);
  }
  .store-detail.desktop {
    max-width: 600px;
  }
  .detail-header {
    display: flex;
    align-items: center;
    gap: var(--space-md);
    margin-bottom: var(--space-md);
  }
  .detail-header h3 {
    margin: 0;
    font-size: 1.125rem;
    color: var(--color-text-bright);
  }
  .trust-badge {
    display: inline-flex;
    align-items: center;
    gap: var(--space-xs);
    font-size: 0.75rem;
    font-weight: 500;
  }
  .detail-desc {
    margin: 0 0 var(--space-md);
    font-size: 0.875rem;
    color: var(--color-muted);
    line-height: 1.6;
  }
  .detail-keywords {
    display: flex;
    flex-wrap: wrap;
    gap: var(--space-xs);
    margin-bottom: var(--space-md);
  }
  .keyword-chip {
    padding: 2px 8px;
    border-radius: 8px;
    background: var(--color-surface-hover);
    color: var(--color-muted);
    font-size: 0.6875rem;
  }
  .detail-meta {
    display: flex;
    align-items: center;
    gap: var(--space-sm);
    font-size: 0.8125rem;
    color: var(--color-muted);
    margin-bottom: var(--space-lg);
    font-family: var(--font-mono);
  }
  .meta-sep { opacity: 0.4; }

  /* Preview iframe */
  .detail-preview {
    aspect-ratio: 3 / 2;
    border-radius: var(--radius-md);
    overflow: hidden;
    border: 1px solid var(--color-border);
    margin-bottom: var(--space-lg);
    background: var(--color-surface);
  }
  .detail-preview iframe {
    width: 100%;
    height: 100%;
    border: none;
  }

  /* README */
  .readme-loading {
    display: flex;
    align-items: center;
    gap: var(--space-sm);
    color: var(--color-muted);
    font-size: 0.8125rem;
    margin-bottom: var(--space-lg);
  }
  .detail-readme {
    margin-bottom: var(--space-lg);
  }
  .detail-readme h4 {
    margin: 0 0 var(--space-sm);
    font-size: 0.875rem;
    color: var(--color-text);
  }
  .readme-text {
    margin: 0;
    padding: var(--space-md);
    background: var(--color-surface);
    border-radius: var(--radius-md);
    border: 1px solid var(--color-border);
    font-size: 0.8125rem;
    color: var(--color-muted);
    line-height: 1.6;
    white-space: pre-wrap;
    word-break: break-word;
    overflow-y: auto;
    max-height: 400px;
  }

  /* Repo link */
  .detail-repo {
    margin-bottom: var(--space-lg);
  }
  .repo-link {
    display: inline-flex;
    align-items: center;
    gap: var(--space-xs);
    color: var(--color-accent);
    text-decoration: none;
    font-size: 0.8125rem;
    cursor: pointer;
    transition: opacity var(--transition);
  }
  .repo-link:hover { opacity: 0.8; }

  /* Install button */
  .detail-install {
    margin-bottom: var(--space-xl);
  }
  .btn-install, .btn-installed, .btn-installing {
    display: inline-flex;
    align-items: center;
    gap: var(--space-xs);
    padding: var(--space-sm) var(--space-lg);
    border-radius: var(--radius-md);
    font-size: 0.875rem;
    font-weight: 500;
    cursor: pointer;
    transition: all var(--transition);
    border: none;
  }
  .btn-install {
    background: var(--color-accent);
    color: var(--color-bg);
  }
  .btn-install:hover { opacity: 0.9; }
  .btn-installed {
    background: var(--color-surface);
    color: var(--color-muted);
    cursor: default;
  }
  .btn-installing {
    background: var(--color-surface);
    color: var(--color-muted);
    cursor: not-allowed;
  }

  :global(.spin) {
    animation: spin 1s linear infinite;
  }
  @keyframes spin {
    from { transform: rotate(0deg); }
    to { transform: rotate(360deg); }
  }
</style>
