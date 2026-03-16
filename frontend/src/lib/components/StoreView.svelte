<script lang="ts">
  import { onMount } from 'svelte';
  import { fly } from 'svelte/transition';
  import { cubicOut } from 'svelte/easing';
  import { ArrowLeft, Search, X, Loader, ExternalLink, Download, Check, RefreshCw, ShieldCheck, Shield, Users, ShieldOff, Star, Settings, ChevronDown } from 'lucide-svelte';
  import { goto } from '$app/navigation';
  import type { RegistryItem, Registry, RegistryCategory, StatsMap } from '$lib/api/types';
  import { marked } from 'marked';
  import DOMPurify from 'dompurify';
  import Fuse from 'fuse.js';
  import { installState } from '$lib/stores/install-state.svelte';
  import Toast from '$lib/components/Toast.svelte';

  interface Props {
    mode: 'desktop' | 'web';
    registryUrl: string;
    mockRegistryUrl?: string;
    title: string;
    installFn?: (item: RegistryItem) => Promise<void>;
    installedVersions?: Map<string, string>; // Changed from installedNames
    previewUrl?: (name: string) => string;
    readmeUrl?: (name: string) => string;
    backRoute?: string;
    communityEnabled?: boolean;
    statsUrl?: string;
    onInstallSuccess?: (name: string) => void;
    initialSelectedId?: string | null;
  }

  let {
    mode,
    registryUrl,
    mockRegistryUrl,
    title,
    installFn,
    installedVersions = new Map<string, string>(),
    previewUrl,
    readmeUrl,
    backRoute,
    communityEnabled = true,
    statsUrl,
    onInstallSuccess,
    initialSelectedId = null,
  }: Props = $props();

  // --- Internal registry state ---
  let registry = $state<Registry | null>(null);
  let loading = $state(false);
  let error = $state<string | null>(null);

  function resolvedUrl(): string {
    if (mode === 'web') return registryUrl;
    const isDev = import.meta.env.DEV || import.meta.env.VITE_MOCK === '1';
    return isDev && mockRegistryUrl ? mockRegistryUrl : registryUrl;
  }

  async function loadRegistry(force = false) {
    if (registry && !force) return;
    loading = true;
    error = null;
    try {
      const res = await fetch(resolvedUrl());
      if (!res.ok) throw new Error(`${res.status}`);
      registry = await res.json();
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
    } finally {
      loading = false;
    }
  }

  async function refreshRegistry() {
    return loadRegistry(true);
  }

  // --- UI state ---
  let searchQuery = $state('');
  let activeCategory = $state<string | null>(null);
  let activeTrust = $state<string | null>(null);
  let selectedId = $state<string | null>(initialSelectedId);
  let readmeContent = $state('');
  let readmeLoading = $state(false);
  let previewWidth = $state(0);
  let currentPage = $state(1);
  let pageSize = $state(24);
  let _toastState = $state<{ message: string; type: 'success' | 'error'; onRetry?: () => void } | null>(null);

  type SortOption = 'latest' | 'stars' | 'downloads';
  let sortBy = $state<SortOption>('latest');
  let sortOpen = $state(false);
  let sortWrapperEl: HTMLDivElement;
  const sortLabels: Record<SortOption, string> = { latest: '最新发布', stars: '最多星标', downloads: '最多下载' };

  let statsMap = $state<StatsMap>({});

  let detailEl = $state<HTMLElement | null>(null);
  let showScrollTop = $state(false);

  function onDetailScroll() {
    if (!detailEl) return;
    showScrollTop = detailEl.scrollTop > 300;
  }

  function scrollToTop() {
    detailEl?.scrollTo({ top: 0, behavior: 'smooth' });
  }

  async function loadStats() {
    if (!statsUrl) return;
    try {
      const res = await fetch(statsUrl);
      if (res.ok) statsMap = await res.json();
    } catch { /* silent */ }
  }

  function formatCount(n: number | undefined): string {
    if (n == null) return '';
    if (n >= 1000) return (n / 1000).toFixed(1).replace(/\.0$/, '') + 'k';
    return String(n);
  }

  let showExactStats = $state(false);

  // Category scroll state
  let categoryScrollEl = $state<HTMLElement | null>(null);
  let canScrollLeft = $state(false);
  let canScrollRight = $state(false);

  function updateScrollState() {
    if (!categoryScrollEl) return;
    canScrollLeft = categoryScrollEl.scrollLeft > 4;
    canScrollRight = categoryScrollEl.scrollLeft < categoryScrollEl.scrollWidth - categoryScrollEl.clientWidth - 4;
  }

  function scrollCategories(dir: 'left' | 'right') {
    categoryScrollEl?.scrollBy({ left: dir === 'left' ? -200 : 200, behavior: 'smooth' });
  }

  $effect(() => {
    if (!categoryScrollEl) return;
    updateScrollState();
    categoryScrollEl.addEventListener('scroll', updateScrollState, { passive: true });
    const ro = new ResizeObserver(updateScrollState);
    ro.observe(categoryScrollEl);
    return () => { categoryScrollEl?.removeEventListener('scroll', updateScrollState); ro.disconnect(); };
  });

  // Card grid auto page size
  let cardGridEl = $state<HTMLElement | null>(null);

  function computePageSize() {
    if (mode === 'web' || !cardGridEl) return;
    const style = getComputedStyle(cardGridEl);
    const gap = parseFloat(style.gap) || 12;
    const colWidth = 200 + gap;
    const cols = Math.max(1, Math.floor((cardGridEl.clientWidth + gap) / colWidth));
    const firstCard = cardGridEl.querySelector('.tpl-card') as HTMLElement | null;
    const cardH = firstCard ? firstCard.offsetHeight : 120;
    const rowHeight = cardH + gap;
    const rows = Math.max(1, Math.round((cardGridEl.clientHeight + gap) / rowHeight));
    const auto = cols * rows;
    if (auto > 0 && auto !== pageSize) pageSize = auto;
  }

  $effect(() => {
    if (!cardGridEl) return;
    computePageSize();
    const ro = new ResizeObserver(computePageSize);
    ro.observe(cardGridEl);
    return () => ro.disconnect();
  });

  // Re-compute after data loads and cards render
  $effect(() => {
    if (mode === 'web' && registry) {
      pageSize = registry.templates.length || 100;
    } else if (registry && cardGridEl) {
      requestAnimationFrame(() => computePageSize());
    }
  });

  // v2: derive categories from templates' category field when registry.categories is absent
  let categories = $derived.by(() => {
    if (!registry) return [];
    if (registry.categories?.length) return registry.categories;
    const seen = new Set<string>();
    return registry.templates
      .map(t => t.category)
      .filter((c): c is string => !!c && !seen.has(c) && (seen.add(c), true))
      .map(c => ({ id: c, label: { zh: c, en: c } }));
  });

  const trustBadge = {
    official: { label: '官方', cls: 'trust-official', color: '#3b82f6', icon: ShieldCheck },
    verified: { label: '已验证', cls: 'trust-verified', color: '#22c55e', icon: Shield },
    community: { label: '社区', cls: 'trust-community', color: '', icon: Users },
    unverified: { label: '未验证', cls: 'trust-unverified', color: '#e0af68', icon: ShieldOff },
  } as const;

  let fuse = $derived(registry ? new Fuse(registry.templates, {
    keys: [
      { name: 'displayName', weight: 2 },
      { name: 'name', weight: 1.5 },
      { name: 'description', weight: 1 },
      { name: 'author', weight: 1 },
      { name: 'category', weight: 0.8 },
      { name: 'keywords', weight: 1.2 },
    ],
    threshold: 0.4,
    ignoreLocation: true,
  }) : null);

  let filteredTemplates = $derived.by(() => {
    if (!registry) return [];
    const q = searchQuery.trim();
    let results = q && fuse
      ? fuse.search(q).map(r => r.item)
      : registry.templates;
    return results.filter(tpl => {
      const matchesCategory = !activeCategory || tpl.category === activeCategory;
      const matchesTrust = !activeTrust || tpl.trust === activeTrust;
      // 社区模板开关关闭时，只显示 official + verified
      const matchesCommunity = communityEnabled ||
        tpl.trust === 'official' || tpl.trust === 'verified';
      return matchesCategory && matchesTrust && matchesCommunity;
    });
  });

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

  // Reset page when filters or page size change
  $effect(() => {
    searchQuery; activeCategory; activeTrust; pageSize; sortBy;
    currentPage = 1;
  });

  let totalPages = $derived(Math.max(1, Math.ceil(sortedTemplates.length / pageSize)));
  let pagedTemplates = $derived(sortedTemplates.slice((currentPage - 1) * pageSize, currentPage * pageSize));

  // Which trust levels have visible templates (respecting communityEnabled)
  let visibleTrustLevels = $derived.by(() => {
    if (!registry) return new Set<string>();
    const levels = new Set<string>();
    for (const tpl of registry.templates) {
      if (!communityEnabled && tpl.trust !== 'official' && tpl.trust !== 'verified') continue;
      levels.add(tpl.trust);
    }
    return levels;
  });

  let selectedTemplate = $derived(
    registry?.templates.find(t => t.name === selectedId) ?? null
  );

  let selectedBadge = $derived(
    selectedTemplate ? trustBadge[selectedTemplate.trust] : null
  );

  function isInstalled(name: string): boolean {
    return installedVersions.has(name);
  }

  function hasUpdate(name: string, latestVersion: string): boolean {
    const installedVersion = installedVersions.get(name);
    if (!installedVersion) return false;
    return installedVersion !== latestVersion;
  }

  const errorMessages: Record<string, string> = {
    network_error: '网络连接失败，请检查网络后重试',
    not_found: '模板不存在',
    checksum_mismatch: '文件校验失败，可能已损坏，请重试',
    server_error: '服务器暂时不可用，请稍后重试',
  };

  async function handleInstall(tpl: RegistryItem) {
    if (!installFn || installState.isInstalling(tpl.name)) return;

    installState.setInstalling(tpl.name);

    try {
      await installFn(tpl);
      installState.setInstalled(tpl.name);
      onInstallSuccess?.(tpl.name);
      // Success toast is optional - button shows state anyway
    } catch (err) {
      // Parse error response from API
      let errorMessage = '安装失败，请重试';
      let errorType = 'server_error';

      if (err instanceof Error) {
        try {
          const errorData = JSON.parse(err.message);
          if (errorData.error_type && errorMessages[errorData.error_type]) {
            errorMessage = errorMessages[errorData.error_type];
            errorType = errorData.error_type;
          }
        } catch {
          // Not JSON, use default message
        }
      }

      // Show error toast with retry
      _toastState = {
        message: errorMessage,
        type: 'error',
        onRetry: () => {
          _toastState = null;
          handleInstall(tpl); // Retry
        },
      };

      // Reset state after toast (per user decision: button returns to "Install" state)
      setTimeout(() => {
        installState.reset(tpl.name);
      }, 3500); // After toast duration + buffer

      console.error('[StoreView] install failed:', err);
    }
  }

  async function loadReadme(name: string) {
    if (!readmeUrl) return;
    readmeLoading = true;
    readmeContent = '';
    try {
      const url = mode === 'desktop' && (import.meta.env.DEV || import.meta.env.VITE_MOCK === '1')
        ? '/mock/README.md'
        : readmeUrl(name);
      const res = await fetch(url);
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
    if (selectedId === id) {
      selectedId = null;
      return;
    }
    selectedId = id;
    loadReadme(id);
  }

  function openExternal(url: string) {
    if (mode === 'desktop' && (window as any).runtime?.BrowserOpenURL) {
      (window as any).runtime.BrowserOpenURL(url);
    } else {
      window.open(url, '_blank', 'noopener,noreferrer');
    }
  }

  function getRepoUrl(tpl: RegistryItem): string {
    if (tpl.repository) return tpl.repository;
    if (tpl.repo) return `https://github.com/${tpl.repo}`;
    return '';
  }

  const renderer = new marked.Renderer();
  renderer.link = ({ text }) => text;
  renderer.image = ({ text }) => text ? `[${text}]` : '';
  marked.setOptions({ gfm: true, breaks: true, renderer });

  function renderMarkdown(src: string): string {
    return marked.parse(src, { async: false }) as string;
  }

  function handleSortOutsideClick(e: PointerEvent) {
    if (sortOpen && sortWrapperEl && !sortWrapperEl.contains(e.target as Node)) {
      sortOpen = false;
    }
  }

  onMount(() => {
    loadRegistry();
    loadStats();
    document.addEventListener('pointerdown', handleSortOutsideClick, true);
    return () => document.removeEventListener('pointerdown', handleSortOutsideClick, true);
  });
</script>

<div class="page" class:web-mode={mode === 'web'}>
  {#if mode === 'desktop'}
    <div class="drag-region" style="--wails-draggable:drag"></div>
  {/if}
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

  {#if loading && !registry}
    <div class="store-empty">
      <Loader size={24} class="spin" />
      <p>加载中…</p>
    </div>
  {:else if error && !registry}
    <div class="store-empty">
      <p class="error-text">加载失败：{error}</p>
      <button class="btn-retry" onclick={() => refreshRegistry()}>重试</button>
    </div>
  {:else if registry}
    <!-- Filter Toolbar -->
    <div class="filter-toolbar">
      <!-- Row 1: Search + Sort -->
      <div class="search-sort-row">
        <div class="search-box">
          <span class="search-icon"><Search size={14} /></span>
          <input
            type="text"
            class="search-input"
            placeholder="搜索名称、描述或标签…"
            bind:value={searchQuery}
          />
          {#if searchQuery}
            <button class="search-clear" onclick={() => searchQuery = ''}>
              <X size={12} />
            </button>
          {/if}
        </div>
        <div class="sort-wrapper" bind:this={sortWrapperEl}>
          <button
            class="sort-trigger"
            class:open={sortOpen}
            onclick={() => sortOpen = !sortOpen}
            aria-haspopup="listbox"
            aria-expanded={sortOpen}
          >
            <span class="sort-label">{sortLabels[sortBy]}</span>
            <ChevronDown size={12} />
          </button>
          {#if sortOpen}
            <div class="sort-dropdown" role="listbox" transition:fly={{ y: -4, duration: 150, easing: cubicOut }}>
              {#each (['latest', 'stars', 'downloads'] as const) as opt (opt)}
                <button
                  class="sort-option"
                  class:selected={sortBy === opt}
                  role="option"
                  aria-selected={sortBy === opt}
                  onclick={() => { sortBy = opt; sortOpen = false; }}
                >
                  {sortLabels[opt]}
                </button>
              {/each}
            </div>
          {/if}
        </div>
      </div>
      <!-- Row 2: Trust Toggles (left) + Categories (right, scrollable) -->
      <div class="controls-row">
        {#if visibleTrustLevels.size > 1}
        <div class="trust-toggles">
          {#each Object.entries(trustBadge) as [key, badge] (key)}
            {#if visibleTrustLevels.has(key)}
              {@const BadgeIcon = badge.icon}
              <button
                class="trust-toggle"
                class:active={activeTrust === key}
                style="--toggle-color:{badge.color || 'var(--color-muted)'}"
                onclick={() => activeTrust = activeTrust === key ? null : key}
                title={badge.label}
              >
                <span class="trust-dot"></span>
                <BadgeIcon size={13} />
                <span class="trust-label">{badge.label}</span>
              </button>
            {/if}
          {/each}
        </div>
        <div class="controls-sep"></div>
        {/if}
        <div class="category-bar">
          {#if canScrollLeft}
            <button class="scroll-arrow scroll-arrow-left" onclick={() => scrollCategories('left')} aria-label="向左滚动">‹</button>
          {/if}
          <div class="category-scroll" bind:this={categoryScrollEl}>
            <button class="cat-chip" class:active={!activeCategory} onclick={() => activeCategory = null}>全部</button>
            {#each categories as cat (cat.id)}
              <button class="cat-chip" class:active={activeCategory === cat.id} onclick={() => activeCategory = activeCategory === cat.id ? null : cat.id}>{cat.label.zh}</button>
            {/each}
          </div>
          {#if canScrollRight}
            <button class="scroll-arrow scroll-arrow-right" onclick={() => scrollCategories('right')} aria-label="向右滚动">›</button>
          {/if}
        </div>
      </div>
    </div>

    {#if selectedId && selectedTemplate}
      <!-- Master-Detail View -->
      <div class="master-detail">
        <nav class="store-nav">
          {#each sortedTemplates as tpl (tpl.name)}
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

        <div class="store-detail" bind:this={detailEl} onscroll={onDetailScroll}>
          <!-- Header -->
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
          {#if selectedTemplate.trust === 'official' || selectedTemplate.trust === 'verified'}
            {#if previewUrl}
              <div
                class="detail-preview"
                bind:clientWidth={previewWidth}
                style="height:{previewWidth * 800 / 1200}px"
              >
                <iframe
                  src={previewUrl(selectedTemplate.name)}
                  sandbox="allow-scripts allow-same-origin"
                  loading="lazy"
                  title="预览"
                  style="transform:scale({previewWidth / 1200})"
                ></iframe>
              </div>
            {/if}
          {:else}
            <div class="detail-preview-placeholder">
              <Shield size={24} />
              <span>社区模板暂不提供预览</span>
            </div>
          {/if}

          <!-- README -->
          {#if readmeUrl}
            {#if readmeLoading}
              <div class="readme-loading">
                <Loader size={16} class="spin" />
                <span>加载 README…</span>
              </div>
            {:else if readmeContent}
              <div class="detail-readme">
                <h4>README</h4>
                <!-- SEC-34: Sanitize rendered markdown to prevent XSS -->
                <div class="readme-body">{@html DOMPurify.sanitize(renderMarkdown(readmeContent))}</div>
              </div>
            {/if}
          {/if}

          <!-- Repository -->
          <div class="detail-repo">
            {#if getRepoUrl(selectedTemplate)}
            <a
              href={getRepoUrl(selectedTemplate)}
              onclick={(e) => { e.preventDefault(); openExternal(getRepoUrl(selectedTemplate!)); }}
              class="repo-link"
            >
              查看源码
              <ExternalLink size={12} />
            </a>
            {/if}
          </div>

          <!-- Bottom actions -->
          <div class="detail-actions">
            <div class="actions-left">
              {#if mode === 'desktop' && installFn}
                {#if installState.isInstalling(selectedTemplate.name)}
                  <button class="btn-installing" disabled>
                    <span class="status-dot"></span><span>安装中...</span>
                  </button>
                {:else if isInstalled(selectedTemplate.name) && hasUpdate(selectedTemplate.name, selectedTemplate.version)}
                  <button class="btn-install" onclick={() => handleInstall(selectedTemplate!)}>
                    <RefreshCw size={14} /><span>更新</span>
                  </button>
                {:else if !isInstalled(selectedTemplate.name)}
                  <button class="btn-install" onclick={() => handleInstall(selectedTemplate!)}>
                    <Download size={14} /><span>安装</span>
                  </button>
                {/if}
              {:else if mode === 'web'}
                <button class="btn-install" onclick={() => {
                  const url = `presto://install/${selectedTemplate!.name}`;
                  if (window.parent !== window) {
                    window.parent.postMessage({ type: 'presto-open-template', url }, '*');
                  } else {
                    window.location.href = url;
                  }
                }}>
                  <Download size={14} /><span>在 Presto 中打开</span>
                </button>
              {/if}
              {#if mode === 'desktop' && isInstalled(selectedTemplate.name)}
                <button
                  class="btn-manage-lg"
                  onclick={() => goto(`/settings?panel=tpl-manage&focus=${selectedTemplate.name}`)}
                >
                  <Settings size={14} /><span>管理</span>
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
        </div>
      </div>
    {:else}
      <!-- Card Grid View -->
      {#if sortedTemplates.length === 0}
        <div class="store-empty">
          <p>{searchQuery ? '没有匹配的结果' : '暂无可用内容'}</p>
        </div>
      {:else}
        <div class="card-grid" bind:this={cardGridEl}>
          {#each pagedTemplates as tpl (tpl.name)}
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
            </button>
          {/each}
        </div>
        <!-- Pagination -->
        {#if mode !== 'web'}
        <div class="pagination">
          <span class="page-info">{sortedTemplates.length} 项，第 {currentPage}/{totalPages} 页</span>
          {#if totalPages > 1}
            <div class="page-controls">
              <button class="page-btn" disabled={currentPage <= 1} onclick={() => currentPage--}>&lsaquo;</button>
              {#each Array.from({length: totalPages}, (_, i) => i + 1) as p}
                {#if p === 1 || p === totalPages || (p >= currentPage - 3 && p <= currentPage + 3)}
                  <button class="page-btn" class:active={currentPage === p} onclick={() => currentPage = p}>{p}</button>
                {:else if p === currentPage - 4 || p === currentPage + 4}
                  <span class="page-ellipsis">…</span>
                {/if}
              {/each}
              <button class="page-btn" disabled={currentPage >= totalPages} onclick={() => currentPage++}>&rsaquo;</button>
            </div>
          {/if}
        </div>
        {/if}
      {/if}
    {/if}
  {/if}

  {#if _toastState}
    <Toast
      message={_toastState.message}
      type={_toastState.type}
      duration={3000}
      onRetry={_toastState.onRetry}
    />
  {/if}
</div>

<style>
  .page {
    padding: var(--space-xl);
    padding-top: 48px;
    height: 100%;
    display: flex;
    flex-direction: column;
    position: relative;
  }
  .page.web-mode {
    padding-top: var(--space-xl);
    height: auto;
    overflow: visible;
  }
  .page.web-mode .store-detail {
    overflow: visible;
  }
  .page.web-mode .card-grid {
    overflow: visible;
  }
  .drag-region {
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    height: 48px;
    z-index: 1;
  }
  h2 {
    margin: 0;
    font-size: 1.125rem;
    font-family: var(--font-ui);
    color: var(--color-text-bright);
  }
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

  /* Filter Toolbar */
  .filter-toolbar {
    display: flex;
    flex-direction: column;
    gap: var(--space-sm);
    margin-bottom: var(--space-xl);
    flex-shrink: 0;
  }
  .search-box {
    position: relative;
    display: flex;
    align-items: center;
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    transition: border-color 250ms ease, box-shadow 250ms ease;
  }
  .search-box:focus-within {
    border-color: var(--color-accent-border);
    box-shadow: 0 0 0 3px rgba(122, 162, 247, 0.08);
  }
  .search-icon {
    display: flex;
    align-items: center;
    padding-left: var(--space-md);
    color: var(--color-muted);
    transition: color 200ms ease;
    flex-shrink: 0;
  }
  .search-box:focus-within .search-icon { color: var(--color-accent); }
  .search-input {
    flex: 1;
    background: none;
    border: none;
    outline: none;
    color: var(--color-text-bright);
    font-family: var(--font-ui);
    font-size: 13px;
    padding: 9px var(--space-md) 9px var(--space-sm);
    line-height: 1;
  }
  .search-input::placeholder { color: var(--color-muted); }
  .search-clear {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 20px;
    height: 20px;
    margin-right: var(--space-sm);
    border-radius: 50%;
    border: none;
    background: var(--color-surface-hover);
    color: var(--color-muted);
    cursor: pointer;
    transition: background 150ms ease, color 150ms ease, transform 120ms ease;
    flex-shrink: 0;
  }
  .search-clear:hover { background: rgba(255, 255, 255, 0.1); color: var(--color-text); transform: scale(1.1); }
  .search-clear:active { transform: scale(0.9); }

  /* Controls Row */
  .controls-row {
    display: flex;
    align-items: center;
    gap: var(--space-md);
  }

  /* Category Bar */
  .category-bar {
    position: relative;
    flex: 1;
    min-width: 0;
    isolation: isolate;
  }
  .category-scroll {
    display: flex;
    gap: var(--space-sm);
    overflow-x: auto;
    scroll-behavior: smooth;
    padding: 2px var(--space-sm);
    scrollbar-width: none;
    -ms-overflow-style: none;
  }
  .category-scroll::-webkit-scrollbar { display: none; }
  .cat-chip {
    flex: 0 0 auto;
    display: inline-flex;
    align-items: center;
    height: 30px;
    padding: 0 var(--space-md);
    border: 1px solid var(--color-border);
    border-radius: 999px;
    background: transparent;
    color: var(--color-muted);
    font-family: var(--font-ui);
    font-size: 12.5px;
    font-weight: 500;
    white-space: nowrap;
    cursor: pointer;
    transition: color var(--transition), background var(--transition), border-color var(--transition);
    user-select: none;
  }
  .cat-chip:hover { color: var(--color-text); background: var(--color-surface-hover); border-color: rgba(255,255,255,0.1); }
  .cat-chip.active { color: var(--color-accent); background: var(--color-accent-bg); border-color: var(--color-accent-border); }
  .scroll-arrow {
    position: absolute;
    top: 50%;
    z-index: 2;
    display: grid;
    place-items: center;
    width: 24px;
    height: 24px;
    border: 1px solid var(--color-border);
    border-radius: 50%;
    background: var(--color-surface);
    color: var(--color-muted);
    font-size: 14px;
    font-family: var(--font-ui);
    cursor: pointer;
    transform: translateY(-50%);
    transition: background var(--transition), color var(--transition), border-color var(--transition);
  }
  .scroll-arrow:hover { background: var(--color-accent-bg); color: var(--color-accent); border-color: var(--color-accent-border); }
  .scroll-arrow-left { left: 0; }
  .scroll-arrow-right { right: 0; }
  .scroll-arrow::before {
    content: "";
    position: absolute;
    top: 50%;
    width: 48px;
    height: calc(100% + 16px);
    transform: translateY(-50%);
    pointer-events: none;
    z-index: -1;
    border-radius: var(--radius-md);
  }
  .scroll-arrow-left::before { left: -4px; background: linear-gradient(to right, var(--color-bg) 40%, transparent 100%); }
  .scroll-arrow-right::before { right: -4px; background: linear-gradient(to left, var(--color-bg) 40%, transparent 100%); }

  /* Separator */
  .controls-sep {
    width: 1px;
    height: 20px;
    background: var(--color-border);
    flex-shrink: 0;
  }

  /* Trust Toggles */
  .trust-toggles {
    display: flex;
    align-items: center;
    gap: 2px;
  }
  .trust-toggle {
    display: flex;
    align-items: center;
    gap: var(--space-xs);
    padding: 5px var(--space-sm);
    border: 1px solid transparent;
    border-radius: var(--radius-sm);
    background: none;
    color: var(--color-muted);
    font-family: var(--font-ui);
    font-size: 12px;
    font-weight: 500;
    cursor: pointer;
    white-space: nowrap;
    line-height: 1;
    transition: color 200ms ease, background 200ms ease, border-color 200ms ease, box-shadow 200ms ease;
    user-select: none;
  }
  .trust-dot {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: var(--toggle-color);
    opacity: 0.35;
    flex-shrink: 0;
    transition: opacity 250ms ease, transform 250ms ease, box-shadow 250ms ease;
  }
  .trust-toggle:hover { color: var(--color-text); background: var(--color-surface); }
  .trust-toggle:hover .trust-dot { opacity: 0.7; transform: scale(1.2); }
  .trust-toggle:active { transform: scale(0.97); }
  .trust-toggle.active {
    color: var(--color-text-bright);
    background: color-mix(in srgb, var(--toggle-color) 10%, transparent);
    border-color: color-mix(in srgb, var(--toggle-color) 25%, transparent);
  }
  .trust-toggle.active .trust-dot {
    opacity: 1;
    transform: scale(1.3);
    box-shadow: 0 0 4px color-mix(in srgb, var(--toggle-color) 50%, transparent);
  }
  .trust-label {
    transition: opacity 150ms ease;
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
    grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
    align-content: start;
    gap: var(--space-md);
    overflow-y: auto;
    flex: 1;
  }

  /* Pagination */
  .pagination {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: var(--space-md) 0 0;
    flex-shrink: 0;
  }
  .page-info {
    font-size: 0.75rem;
    color: var(--color-muted);
  }
  .page-controls {
    display: flex;
    align-items: center;
    gap: 2px;
  }
  .page-btn {
    min-width: 28px;
    height: 28px;
    display: flex;
    align-items: center;
    justify-content: center;
    border: 1px solid var(--color-border);
    border-radius: var(--radius-sm);
    background: var(--color-surface);
    color: var(--color-muted);
    font-size: 0.75rem;
    font-family: var(--font-ui);
    cursor: pointer;
    transition: all var(--transition);
  }
  .page-btn:hover:not(:disabled) { color: var(--color-text); border-color: var(--color-accent-border); }
  .page-btn.active { background: var(--color-accent-bg); color: var(--color-accent); border-color: var(--color-accent-border); }
  .page-btn:disabled { opacity: 0.3; cursor: not-allowed; }
  .page-ellipsis { color: var(--color-muted); font-size: 0.75rem; padding: 0 4px; }
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
  .search-sort-row {
    display: flex;
    align-items: center;
    gap: var(--space-md);
  }
  .search-sort-row .search-box {
    flex: 1;
  }
  .sort-wrapper {
    position: relative;
    flex-shrink: 0;
    align-self: stretch;
  }
  .sort-trigger {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    height: 100%;
    padding: 9px var(--space-md);
    background: var(--color-surface);
    color: var(--color-text);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    font-size: 13px;
    font-family: var(--font-ui);
    cursor: pointer;
    transition: border-color var(--transition);
    box-sizing: border-box;
    white-space: nowrap;
  }
  .sort-trigger:hover {
    border-color: var(--color-muted);
  }
  .sort-trigger.open {
    border-color: var(--color-accent);
  }
  .sort-trigger :global(svg) {
    transition: transform 150ms ease;
    flex-shrink: 0;
  }
  .sort-trigger.open :global(svg) {
    transform: rotate(180deg);
  }
  .sort-label {
    flex: 1;
    text-align: left;
  }
  .sort-dropdown {
    position: absolute;
    top: calc(100% + 4px);
    right: 0;
    min-width: 100%;
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    box-shadow: var(--shadow-md);
    z-index: 100;
    display: flex;
    flex-direction: column;
    overflow: hidden;
    padding: var(--space-xs) 0;
  }
  .sort-option {
    display: flex;
    align-items: center;
    width: 100%;
    text-align: left;
    padding: 6px 12px;
    background: none;
    border: none;
    color: var(--color-text);
    font-size: 12px;
    font-family: var(--font-ui);
    cursor: pointer;
    transition: background var(--transition);
    white-space: nowrap;
  }
  .sort-option:hover {
    background: var(--color-surface-hover);
  }
  .sort-option.selected {
    color: var(--color-accent);
    font-weight: 500;
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
    min-width: 0;
    overflow-y: auto;
    padding-right: var(--space-md);
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
    max-width: 100%;
    border-radius: var(--radius-md);
    overflow: hidden;
    border: 1px solid var(--color-border);
    margin-bottom: var(--space-lg);
    background: var(--color-surface);
  }
  .detail-preview iframe {
    width: 1200px;
    height: 800px;
    transform-origin: 0 0;
    border: none;
  }
  .detail-preview-placeholder {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    padding: 48px 16px;
    color: var(--color-muted);
    background: var(--color-surface);
    border-radius: var(--radius-md);
    font-size: 14px;
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
  .readme-body {
    margin: 0;
    padding: var(--space-md);
    background: var(--color-surface);
    border-radius: var(--radius-md);
    border: 1px solid var(--color-border);
    font-size: 0.8125rem;
    color: var(--color-muted);
    line-height: 1.7;
    word-break: break-word;
  }
  .readme-body :global(h3),
  .readme-body :global(h4),
  .readme-body :global(h5),
  .readme-body :global(h6) {
    color: var(--color-text-bright);
    margin: 1em 0 0.5em;
    line-height: 1.3;
  }
  .readme-body :global(h3) { font-size: 1.1em; }
  .readme-body :global(h4) { font-size: 1em; }
  .readme-body :global(h5) { font-size: 0.95em; }
  .readme-body :global(h6) { font-size: 0.9em; }
  .readme-body :global(p) {
    margin: 0.5em 0;
  }
  .readme-body :global(strong) { color: var(--color-text); }
  .readme-body :global(code) {
    padding: 0.15em 0.4em;
    background: var(--color-surface-hover);
    border-radius: 4px;
    font-family: var(--font-mono);
    font-size: 0.9em;
  }
  .readme-body :global(pre) {
    padding: var(--space-sm);
    background: var(--color-surface-hover);
    border-radius: var(--radius-sm);
    overflow-x: auto;
    margin: 0.75em 0;
  }
  .readme-body :global(pre code) {
    padding: 0;
    background: none;
  }
  .readme-body :global(ul) {
    padding-left: 1.5em;
    margin: 0.5em 0;
  }
  .readme-body :global(li) {
    margin: 0.25em 0;
  }
  .readme-body :global(hr) {
    border: none;
    border-top: 1px solid var(--color-border);
    margin: 1em 0;
  }
  .readme-body :global(table) {
    width: 100%;
    border-collapse: collapse;
    margin: 0.75em 0;
    font-size: 0.8125rem;
  }
  .readme-body :global(th),
  .readme-body :global(td) {
    padding: 0.4em 0.75em;
    border: 1px solid var(--color-border);
    text-align: left;
  }
  .readme-body :global(th) {
    background: var(--color-surface-hover);
    color: var(--color-text);
    font-weight: 600;
  }
  .readme-body :global(tr:nth-child(even)) {
    background: var(--color-surface-hover);
  }
  .readme-body :global(blockquote) {
    margin: 0.75em 0;
    padding: 0.5em 1em;
    border-left: 3px solid var(--color-border);
    color: var(--color-muted);
    background: var(--color-surface-hover);
    border-radius: 0 var(--radius-sm) var(--radius-sm) 0;
  }
  .readme-body :global(ol) {
    padding-left: 1.5em;
    margin: 0.5em 0;
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

  /* Bottom actions */
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
  .btn-manage-lg {
    display: inline-flex;
    align-items: center;
    gap: var(--space-xs);
    padding: var(--space-sm) var(--space-lg);
    border-radius: var(--radius-md);
    font-size: 0.875rem;
    font-weight: 500;
    cursor: pointer;
    transition: all var(--transition);
    border: 1px solid var(--color-border);
    background: var(--color-surface);
    color: var(--color-muted);
  }
  .btn-manage-lg:hover {
    color: var(--color-text);
    border-color: var(--color-accent-border);
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
  .btn-installing .status-dot {
    width: 14px;
    height: 14px;
    border-radius: 50%;
    background: var(--color-accent);
    animation: pulse 1s ease-in-out infinite;
  }
  @keyframes pulse {
    0%, 100% { opacity: 0.4; }
    50% { opacity: 1; }
  }

  :global(.spin) {
    animation: spin 1s linear infinite;
  }
  @keyframes spin {
    from { transform: rotate(0deg); }
    to { transform: rotate(360deg); }
  }

  /* ============================================
     Mobile Responsive (< 768px)
     Card grid: single column full-width
     Master-detail: hide sidebar, detail full-screen
     ============================================ */
  @media (max-width: 767px) {
    .page {
      padding: var(--space-md);
    }
    .page.web-mode {
      padding-top: var(--space-md);
      height: auto;
      overflow-x: hidden;
      overflow-y: visible;
    }

    /* Card grid: single column */
    .card-grid {
      grid-template-columns: 1fr;
    }

    /* Master-detail: hide sidebar, detail takes full width */
    .master-detail {
      gap: 0;
    }
    .store-nav {
      display: none;
    }
    .store-detail {
      padding-right: 0;
    }
    .detail-header {
      flex-wrap: wrap;
    }
    .detail-actions {
      flex-wrap: wrap;
    }

    /* Category bar: tighter spacing */
    .filter-toolbar {
      margin-bottom: var(--space-md);
    }
    .controls-row {
      flex-wrap: wrap;
      gap: var(--space-sm);
    }
    .search-sort-row {
      flex-wrap: wrap;
    }

    /* Pagination: compact */
    .pagination {
      flex-wrap: wrap;
      gap: var(--space-sm);
    }
  }
</style>
