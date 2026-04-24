<script lang="ts">
  import { onMount } from 'svelte';
  import { Search, X, Loader, Copy, Check, ShieldCheck, Shield, Users, ShieldOff, ExternalLink, ArrowLeft } from 'lucide-svelte';
  import { goto } from '$app/navigation';
  import type { RegistrySkill, SkillRegistry } from '$lib/api/types';
  import { marked } from 'marked';
  import DOMPurify from 'dompurify';
  import Fuse from 'fuse.js';

  interface Props {
    mode: 'desktop' | 'web';
    registryUrl: string;
    title: string;
    readmeUrl?: (skill: RegistrySkill) => string;
    backRoute?: string;
    installedNames?: Set<string>;
    initialSelectedId?: string | null;
    locale?: 'zh' | 'en';
  }

  let {
    mode,
    registryUrl,
    title,
    readmeUrl,
    backRoute,
    installedNames,
    initialSelectedId = null,
    locale = 'zh',
  }: Props = $props();

  // --- Labels (bilingual) ---
  const labels: Record<string, { zh: string; en: string }> = {
    search: { zh: '搜索技能...', en: 'Search skills...' },
    noResults: { zh: '未找到匹配的技能', en: 'No matching skills found' },
    installCommand: { zh: '安装命令', en: 'Install Command' },
    copy: { zh: '复制', en: 'Copy' },
    copied: { zh: '已复制', en: 'Copied' },
    github: { zh: '查看 GitHub', en: 'View on GitHub' },
    loading: { zh: '加载中...', en: 'Loading...' },
    errorRetry: { zh: '重试', en: 'Retry' },
    empty: { zh: '暂无技能', en: 'No skills available' },
    description: { zh: '简介', en: 'Description' },
    readme: { zh: '使用说明', en: 'Usage Guide' },
    allCategories: { zh: '全部', en: 'All' },
    loadFailed: { zh: '加载失败：', en: 'Load failed: ' },
  };

  function t(key: string): string {
    return labels[key]?.[locale] ?? key;
  }

  const trustBadge: Record<string, { icon: typeof ShieldCheck; label: { zh: string; en: string }; color: string; cls: string }> = {
    official: { icon: ShieldCheck, label: { zh: '官方', en: 'Official' }, color: '#3b82f6', cls: 'trust-official' },
    verified: { icon: Shield, label: { zh: '已验证', en: 'Verified' }, color: '#22c55e', cls: 'trust-verified' },
    community: { icon: Users, label: { zh: '社区', en: 'Community' }, color: '#a78bfa', cls: 'trust-community' },
    unverified: { icon: ShieldOff, label: { zh: '未验证', en: 'Unverified' }, color: '#e0af68', cls: 'trust-unverified' },
  };

  // --- Registry state ---
  let registry = $state<SkillRegistry | null>(null);
  let loading = $state(false);
  let error = $state<string | null>(null);

  async function loadRegistry(force = false) {
    if (registry && !force) return;
    loading = true;
    error = null;
    try {
      const res = await fetch(registryUrl);
      if (!res.ok) throw new Error(`${res.status}`);
      registry = await res.json();
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
    } finally {
      loading = false;
    }
  }

  // --- UI state ---
  let searchQuery = $state('');
  let activeCategory = $state<string | null>(null);
  let activeTrust = $state<string | null>(null);
  let selectedId = $state<string | null>(null);
  $effect(() => { if (initialSelectedId !== null) selectedId = initialSelectedId; });
  let readmeContent = $state('');
  let readmeLoading = $state(false);
  let copied = $state(false);

  // --- Derived data ---
  let categories = $derived(() => {
    if (!registry) return [];
    const cats = new Set<string>();
    for (const s of registry.skills) cats.add(s.category);
    return Array.from(cats);
  });

  let selectedSkill = $derived(() => {
    if (!registry || !selectedId) return null;
    return registry.skills.find(s => s.name === selectedId) ?? null;
  });

  let visibleTrustLevels = $derived(() => {
    if (!registry) return new Set<string>();
    const levels = new Set<string>();
    for (const s of registry.skills) levels.add(s.trust);
    return levels;
  });

  let filteredSkills = $derived(() => {
    if (!registry) return [];
    let skills = registry.skills;

    // Search
    if (searchQuery.trim()) {
      const fuse = new Fuse(skills, {
        keys: ['displayName', 'description', 'keywords', 'author'],
        threshold: 0.4,
      });
      skills = fuse.search(searchQuery.trim()).map(r => r.item);
    }

    // Category filter
    if (activeCategory) {
      skills = skills.filter(s => s.category === activeCategory);
    }

    // Trust filter
    if (activeTrust) {
      skills = skills.filter(s => s.trust === activeTrust);
    }

    return skills;
  });

  // --- Actions ---
  function selectSkill(name: string) {
    selectedId = name;
  }

  function copyInstallCommand(skill: RegistrySkill) {
    const cmd = `npx skills add --repo ${skill.repo} --path ${skill.path}`;
    navigator.clipboard.writeText(cmd).then(() => {
      copied = true;
      setTimeout(() => { copied = false; }, 2000);
    });
  }

  async function loadReadme(skill: RegistrySkill) {
    const url = readmeUrl?.(skill) ?? `https://raw.githubusercontent.com/${skill.repo}/main/${skill.path}/SKILL.md`;
    readmeLoading = true;
    readmeContent = '';
    try {
      const res = await fetch(url);
      if (!res.ok) throw new Error(`${res.status}`);
      readmeContent = await res.text();
    } catch {
      readmeContent = '';
    } finally {
      readmeLoading = false;
    }
  }

  function renderMarkdown(md: string): string {
    return marked.parse(md, { async: false }) as string;
  }

  function openExternal(url: string) {
    if (mode === 'web' && window.parent !== window) {
      window.parent.postMessage({ type: 'presto-open-skill', url }, '*');
    } else {
      window.open(url, '_blank', 'noopener,noreferrer');
    }
  }

  // --- Effects ---
  $effect(() => {
    if (selectedSkill()) {
      loadReadme(selectedSkill()!);
    } else {
      readmeContent = '';
    }
  });

  onMount(() => {
    loadRegistry();
  });
</script>

{#if loading && !registry}
  <div class="page" class:web-mode={mode === 'web'}>
    <div class="store-empty">
      <Loader size={24} class="spin" />
      <p>{t('loading')}</p>
    </div>
  </div>
{:else if error && !registry}
  <div class="page" class:web-mode={mode === 'web'}>
    <div class="store-empty">
      <p class="error-text">{t('loadFailed')}{error}</p>
      <button class="btn-retry" onclick={() => loadRegistry(true)}>{t('errorRetry')}</button>
    </div>
  </div>
{:else if registry}
<div class="page" class:web-mode={mode === 'web'}>
  {#if mode === 'desktop'}
    <div class="drag-region" style="--wails-draggable:drag"></div>
  {/if}

  <div class="page-header">
    {#if mode === 'desktop' && backRoute}
      <button class="btn-back" onclick={() => goto(backRoute!)} aria-label="返回">
        <ArrowLeft size={16} />
      </button>
    {/if}
    <nav class="breadcrumb">
      {#if selectedId && selectedSkill()}
        <button class="breadcrumb-link" onclick={() => selectedId = null}>{title}</button>
        <span class="breadcrumb-sep">›</span>
        <span class="breadcrumb-current">{selectedSkill()!.displayName}</span>
      {:else}
        <h2>{title}</h2>
      {/if}
    </nav>
  </div>

  <!-- Filter Toolbar -->
  <div class="filter-toolbar">
    <div class="search-sort-row">
      <div class="search-box">
        <span class="search-icon"><Search size={14} /></span>
        <input
          type="text"
          class="search-input"
          placeholder={t('search')}
          bind:value={searchQuery}
        />
        {#if searchQuery}
          <button class="search-clear" onclick={() => searchQuery = ''}>
            <X size={12} />
          </button>
        {/if}
      </div>
    </div>
    <div class="controls-row">
      {#if visibleTrustLevels().size > 1}
        <div class="trust-toggles">
          {#each Object.entries(trustBadge) as [key, badge] (key)}
            {@const BadgeIcon = badge.icon}
            {#if visibleTrustLevels().has(key)}
              <button
                class="trust-toggle"
                class:active={activeTrust === key}
                style="--toggle-color:{badge.color || 'var(--color-muted)'}"
                onclick={() => activeTrust = activeTrust === key ? null : key}
                title={badge.label[locale]}
              >
                <span class="trust-dot"></span>
                <BadgeIcon size={13} />
                <span class="trust-label">{badge.label[locale]}</span>
              </button>
            {/if}
          {/each}
        </div>
        <div class="controls-sep"></div>
      {/if}
      <div class="category-bar">
        <div class="category-scroll">
          <button class="cat-chip" class:active={!activeCategory} onclick={() => activeCategory = null}>{t('allCategories')}</button>
          {#each categories() as cat (cat)}
            <button class="cat-chip" class:active={activeCategory === cat} onclick={() => activeCategory = activeCategory === cat ? null : cat}>{cat}</button>
          {/each}
        </div>
      </div>
    </div>
  </div>

  {#if selectedId && selectedSkill()}
    {@const skill = selectedSkill()!}
    {@const badge = trustBadge[skill.trust]}
    {@const BadgeIcon = badge.icon}
    <!-- Master-Detail View -->
    <div class="master-detail">
      <nav class="store-nav">
        {#each filteredSkills() as s (s.name)}
          {@const sb = trustBadge[s.trust]}
          <button
            class="nav-skill-item"
            class:active={selectedId === s.name}
            onclick={() => selectSkill(s.name)}
          >
            <span class="nav-skill-name">{s.displayName}</span>
            <span class="nav-trust-dot" style="background:{sb.color}"></span>
          </button>
        {/each}
      </nav>

      <div class="store-detail">
        <!-- Header -->
        <div class="detail-header">
          <div class="detail-title-row">
            <h3>{skill.displayName}</h3>
            <span class="trust-badge {badge.cls}" style={badge.color ? `color:${badge.color}` : ''}>
              <BadgeIcon size={14} />
              {badge.label[locale]}
            </span>
            <span class="detail-version">v{skill.version}</span>
            <span class="detail-author">{skill.author}</span>
          </div>
        </div>

        <!-- Description -->
        <p class="detail-desc">{skill.description}</p>

        <!-- Keywords -->
        {#if skill.keywords.length > 0}
          <div class="detail-keywords">
            {#each skill.keywords as kw (kw)}
              <span class="keyword-chip">{kw}</span>
            {/each}
          </div>
        {/if}

        <!-- Install Command -->
        <div class="install-section">
          <h4>{t('installCommand')}</h4>
          {#if mode === 'desktop'}
            <p class="install-hint">在终端中运行以下命令安装</p>
            {#if installedNames?.has(skill.name)}
              <span class="installed-status">✓ 已安装</span>
            {:else}
              <div class="install-cmd-box">
                <code class="install-cmd">npx skills add --repo {skill.repo} --path {skill.path}</code>
              </div>
            {/if}
          {:else}
            <div class="install-cmd-box">
              <code class="install-cmd">npx skills add --repo {skill.repo} --path {skill.path}</code>
              <button
                class="btn-copy"
                class:copied
                onclick={() => copyInstallCommand(skill)}
                aria-label={t('copy')}
              >
                {#if copied}
                  <Check size={14} />
                  <span>{t('copied')}</span>
                {:else}
                  <Copy size={14} />
                  <span>{t('copy')}</span>
                {/if}
              </button>
            </div>
          {/if}
        </div>

        <!-- SKILL.md Content -->
        {#if readmeLoading}
          <div class="readme-loading">
            <Loader size={16} class="spin" />
            <span>{t('loading')}</span>
          </div>
        {:else if readmeContent}
          <div class="detail-readme">
            <h4>{t('readme')}</h4>
            <!-- SEC-34: Sanitize rendered markdown to prevent XSS -->
            <div class="readme-body">{@html DOMPurify.sanitize(renderMarkdown(readmeContent))}</div>
          </div>
        {/if}

        <!-- GitHub Link -->
        <div class="detail-repo">
          <button
            class="repo-link"
            onclick={() => openExternal(`https://github.com/${skill.repo}`)}
          >
            {t('github')}
            <ExternalLink size={12} />
          </button>
        </div>
      </div>
    </div>
  {:else}
    <!-- Card Grid View -->
    {#if filteredSkills().length === 0}
      <div class="store-empty">
        <p>{searchQuery ? t('noResults') : t('empty')}</p>
      </div>
    {:else}
      <div class="card-grid">
        {#each filteredSkills() as s (s.name)}
          {@const sb = trustBadge[s.trust]}
          {@const SBadgeIcon = sb.icon}
          <button class="skill-card" onclick={() => selectSkill(s.name)}>
            {#if mode === 'desktop' && installedNames?.has(s.name)}
              <span class="installed-badge">已安装</span>
            {/if}
            <div class="card-header">
              <span class="card-name">{s.displayName}</span>
              <span class="card-trust {sb.cls}" style={sb.color ? `color:${sb.color}` : ''}>
                <SBadgeIcon size={12} />
                {sb.label[locale]}
              </span>
            </div>
            <p class="card-desc">{s.description}</p>
            <div class="card-footer">
              {#if s.keywords.length > 0}
                <div class="card-keywords">
                  {#each s.keywords.slice(0, 3) as kw (kw)}
                    <span class="keyword-chip">{kw}</span>
                  {/each}
                </div>
              {/if}
            </div>
          </button>
        {/each}
      </div>
    {/if}
  {/if}
</div>
{/if}

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

  /* Breadcrumb */
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

  /* Page header */
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

  /* Filter Toolbar */
  .filter-toolbar {
    display: flex;
    flex-direction: column;
    gap: var(--space-sm);
    margin-bottom: var(--space-xl);
    flex-shrink: 0;
  }
  .search-sort-row {
    display: flex;
    gap: var(--space-sm);
  }
  .search-box {
    position: relative;
    display: flex;
    align-items: center;
    flex: 1;
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
  .skill-card {
    position: relative;
    display: flex;
    flex-direction: column;
    padding: var(--space-lg);
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    cursor: pointer;
    text-align: left;
    transition: border-color 200ms ease, box-shadow 200ms ease, background 200ms ease;
  }
  .skill-card:hover {
    border-color: var(--color-accent-border);
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
    background: var(--color-surface-hover);
  }
  .card-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-sm);
    margin-bottom: var(--space-sm);
  }
  .card-name {
    font-weight: 600;
    font-size: 0.875rem;
    color: var(--color-text-bright);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .card-trust {
    display: inline-flex;
    align-items: center;
    gap: 3px;
    font-size: 0.6875rem;
    font-weight: 500;
    flex-shrink: 0;
  }
  .card-desc {
    margin: 0 0 var(--space-sm);
    font-size: 0.8125rem;
    color: var(--color-muted);
    line-height: 1.5;
    display: -webkit-box;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }
  .card-footer {
    margin-top: auto;
  }
  .card-keywords {
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
  }
  .keyword-chip {
    padding: 2px 8px;
    border-radius: 8px;
    background: var(--color-surface-hover);
    color: var(--color-muted);
    font-size: 0.6875rem;
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
  .nav-skill-item {
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
  .nav-skill-item:hover {
    color: var(--color-text);
    background: var(--color-surface);
  }
  .nav-skill-item.active {
    color: var(--color-accent);
    background: var(--color-accent-bg);
  }
  .nav-skill-name {
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
  .detail-title-row {
    display: flex;
    align-items: center;
    gap: var(--space-md);
    flex-wrap: wrap;
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
  .detail-version {
    font-size: 0.8125rem;
    color: var(--color-muted);
    font-family: var(--font-mono);
  }
  .detail-author {
    font-size: 0.8125rem;
    color: var(--color-muted);
    font-family: var(--font-mono);
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

  /* Install Command */
  .install-section {
    margin-bottom: var(--space-lg);
  }
  .install-section h4 {
    margin: 0 0 var(--space-sm);
    font-size: 0.875rem;
    color: var(--color-text);
  }
  .install-cmd-box {
    position: relative;
    display: flex;
    align-items: center;
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    padding: var(--space-sm) var(--space-md);
    padding-right: 80px;
  }
  .install-cmd {
    font-family: var(--font-mono);
    font-size: 0.8125rem;
    color: var(--color-text);
    word-break: break-all;
  }
  .btn-copy {
    position: absolute;
    right: var(--space-sm);
    top: 50%;
    transform: translateY(-50%);
    display: inline-flex;
    align-items: center;
    gap: 4px;
    padding: 4px 10px;
    border: 1px solid var(--color-border);
    border-radius: var(--radius-sm);
    background: var(--color-surface-hover);
    color: var(--color-muted);
    font-size: 0.75rem;
    font-family: var(--font-ui);
    cursor: pointer;
    transition: all 200ms ease;
  }
  .btn-copy:hover {
    color: var(--color-text);
    border-color: var(--color-accent-border);
  }
  .btn-copy.copied {
    color: #22c55e;
    border-color: rgba(34, 197, 94, 0.3);
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
    background: none;
    border: none;
    font-size: 0.8125rem;
    cursor: pointer;
    transition: opacity var(--transition);
    padding: 0;
  }
  .repo-link:hover { opacity: 0.8; }

  /* Spin animation */
  :global(.spin) {
    animation: spin 1s linear infinite;
  }
  @keyframes spin {
    from { transform: rotate(0deg); }
    to { transform: rotate(360deg); }
  }

  /* Responsive */
  @media (max-width: 768px) {
    .master-detail {
      flex-direction: column;
    }
    .store-nav {
      width: 100%;
      flex-direction: row;
      overflow-x: auto;
      overflow-y: hidden;
      padding-bottom: var(--space-sm);
    }
    .store-detail {
      padding-right: 0;
    }
  }

  /* Desktop mode enhancements */
  .install-hint {
    color: var(--color-muted);
    font-size: 0.875rem;
    margin-bottom: var(--space-xs);
  }

  .installed-status {
    color: var(--color-success);
    font-weight: 500;
    margin-bottom: var(--space-xs);
    display: inline-block;
  }

  .installed-badge {
    position: absolute;
    top: var(--space-sm);
    right: var(--space-sm);
    background: var(--color-success-bg);
    color: var(--color-success);
    padding: 0 var(--space-xs);
    border-radius: var(--radius-full);
    font-size: 0.75rem;
    line-height: 1;
    font-weight: 500;
    pointer-events: none;
  }
</style>
