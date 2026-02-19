<script lang="ts">
  import { onMount } from 'svelte';
  import { ExternalLink, Shield, Info, BookOpen, ArrowLeft, RefreshCw, Search, Package, Download, Trash2, Loader } from 'lucide-svelte';
  import { goto } from '$app/navigation';
  import { listTemplates, discoverTemplates, installTemplate, deleteTemplate } from '$lib/api/client';
  import type { Template, GitHubRepo } from '$lib/api/types';

  // --- Settings state ---
  let communityEnabled = $state(false);
  let showWarning = $state(false);
  let appVersion = $state('0.1.0');
  let updateInfo = $state<{ hasUpdate: boolean; latestVersion: string; downloadURL: string; releaseURL: string } | null>(null);
  let checking = $state(false);
  let updateError = $state('');
  let activeSection = $state('general');
  let activePanel = $state<'tpl-manage' | 'tpl-search' | null>(null);

  // --- Template state (migrated from /templates) ---
  let installed: Template[] = $state([]);
  let available: GitHubRepo[] = $state([]);
  let tplLoading = $state(false);
  let installing = $state('');
  let tplSearch = $state('');
  let browseLoaded = $state(false);
  let installedLoaded = $state(false);

  let filteredInstalled = $derived(
    installed.filter(tpl => {
      const q = tplSearch.toLowerCase();
      return !q ||
        (tpl.displayName || tpl.name).toLowerCase().includes(q) ||
        tpl.description.toLowerCase().includes(q);
    })
  );

  let filteredAvailable = $derived(
    available.filter(repo => {
      const q = tplSearch.toLowerCase();
      return !q ||
        repo.name.toLowerCase().includes(q) ||
        (repo.description || '').toLowerCase().includes(q);
    })
  );

  const sections = [
    { id: 'general', label: '通用' },
    { id: 'template-dev', label: '模板开发' },
    { id: 'about', label: '关于' },
    { id: 'licenses', label: '开源协议' },
  ];

  let panelTabs = $derived(communityEnabled ? [
    { id: 'tpl-manage' as const, label: '模板管理' },
    { id: 'tpl-search' as const, label: '模板搜索' },
  ] : []);

  declare global {
    interface Window {
      go?: { main: { App: {
        GetVersion: () => Promise<string>;
        CheckForUpdate: () => Promise<{ hasUpdate: boolean; currentVersion: string; latestVersion: string; downloadURL: string; releaseURL: string }>;
      } } };
    }
  }

  function openExternal(url: string) {
    if ((window as any).runtime?.BrowserOpenURL) {
      (window as any).runtime.BrowserOpenURL(url);
    } else {
      window.open(url, '_blank', 'noopener,noreferrer');
    }
  }

  async function checkUpdate() {
    checking = true;
    updateError = '';
    updateInfo = null;
    try {
      if (window.go?.main?.App?.CheckForUpdate) {
        const info = await window.go.main.App.CheckForUpdate();
        updateInfo = info;
      } else {
        const resp = await fetch('https://api.github.com/repos/Presto-io/Presto/releases/latest');
        if (!resp.ok) throw new Error(`GitHub API error: ${resp.status}`);
        const release = await resp.json();
        const latest = (release.tag_name as string).replace(/^v/, '');
        updateInfo = {
          hasUpdate: latest !== appVersion && appVersion !== 'dev',
          latestVersion: latest,
          downloadURL: release.html_url,
          releaseURL: release.html_url,
        };
      }
    } catch (e) {
      updateError = e instanceof Error ? e.message : String(e);
    } finally {
      checking = false;
    }
  }

  onMount(async () => {
    communityEnabled = localStorage.getItem('communityTemplates') === 'true';
    if (window.go?.main?.App?.GetVersion) {
      try {
        appVersion = await window.go.main.App.GetVersion();
      } catch {}
    }
    const content = document.querySelector('.settings-content') as HTMLElement;
    if (content) {
      content.addEventListener('scroll', handleScroll);
    }
  });

  function handleScroll() {
    if (activePanel) return;
    const content = document.querySelector('.settings-content') as HTMLElement;
    if (!content) return;
    const scrollTop = content.scrollTop;
    for (let i = sections.length - 1; i >= 0; i--) {
      const el = content.querySelector(`#section-${sections[i].id}`) as HTMLElement;
      if (el && el.offsetTop - 24 <= scrollTop) {
        activeSection = sections[i].id;
        return;
      }
    }
    activeSection = sections[0].id;
  }

  function scrollTo(id: string) {
    activePanel = null;
    tplSearch = '';
    activeSection = id;
    const content = document.querySelector('.settings-content') as HTMLElement;
    const el = content?.querySelector(`#section-${id}`) as HTMLElement;
    if (el && content) {
      content.scrollTo({ top: el.offsetTop - 16, behavior: 'smooth' });
    }
  }

  function togglePanel(id: 'tpl-manage' | 'tpl-search') {
    if (activePanel === id) {
      activePanel = null;
      tplSearch = '';
    } else {
      activePanel = id;
      activeSection = '';
      tplSearch = '';
      if (id === 'tpl-manage' && !installedLoaded) loadInstalled();
      if (id === 'tpl-search' && !browseLoaded) loadBrowse();
    }
  }

  // --- Toggle ---
  function toggleCommunity() {
    if (!communityEnabled) {
      showWarning = true;
    } else {
      communityEnabled = false;
      activePanel = null;
      tplSearch = '';
      localStorage.setItem('communityTemplates', 'false');
    }
  }

  function confirmCommunity() {
    communityEnabled = true;
    showWarning = false;
    localStorage.setItem('communityTemplates', 'true');
  }

  function cancelCommunity() {
    showWarning = false;
  }

  // --- Template actions ---
  async function loadInstalled() {
    tplLoading = true;
    try {
      installed = (await listTemplates()) ?? [];
      installedLoaded = true;
    } catch {}
    finally { tplLoading = false; }
  }

  async function loadBrowse() {
    tplLoading = true;
    try {
      available = (await discoverTemplates()) ?? [];
      browseLoaded = true;
    } catch {}
    finally { tplLoading = false; }
  }

  async function handleInstall(repo: GitHubRepo) {
    installing = repo.full_name;
    try {
      await installTemplate(repo.owner.login, repo.name);
      installed = await listTemplates();
      installedLoaded = true;
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
    <h2>设置</h2>
  </div>

  <div class="settings-layout">
    <nav class="settings-nav">
      {#each sections as sec (sec.id)}
        <button
          class="nav-item"
          class:active={!activePanel && activeSection === sec.id}
          onclick={() => scrollTo(sec.id)}
        >
          {sec.label}
        </button>
      {/each}
      {#if panelTabs.length > 0}
        <div class="nav-divider"></div>
        {#each panelTabs as tab (tab.id)}
          <button
            class="nav-item"
            class:active={activePanel === tab.id}
            onclick={() => togglePanel(tab.id)}
          >
            {tab.label}
          </button>
        {/each}
      {/if}
    </nav>

    <div class="settings-content-wrapper">
      {#if activePanel}
        <div class="panel-overlay">
          <div class="panel-search">
            <Search size={14} />
            <input
              type="text"
              placeholder={activePanel === 'tpl-manage' ? '搜索已安装模板…' : '搜索社区模板…'}
              bind:value={tplSearch}
            />
          </div>

          {#if activePanel === 'tpl-manage'}
            {#if tplLoading && !installedLoaded}
              <div class="panel-empty">
                <Loader size={24} class="spin" />
                <p>加载中...</p>
              </div>
            {:else if filteredInstalled.length === 0}
              <div class="panel-empty">
                <Package size={32} />
                <p>{tplSearch ? '没有匹配的模板' : '暂无已安装模板'}</p>
              </div>
            {:else}
              <div class="tpl-list">
                {#each filteredInstalled as tpl (tpl.name)}
                  <div class="tpl-row">
                    <div class="tpl-info">
                      <div class="tpl-name-row">
                        <span class="tpl-name">{tpl.displayName || tpl.name}</span>
                        <span class="tpl-version">v{tpl.version}</span>
                        {#if tpl.builtin}
                          <span class="badge-builtin">内置</span>
                        {/if}
                      </div>
                      <p class="tpl-desc">{tpl.description}</p>
                      <span class="tpl-author">{tpl.author}</span>
                    </div>
                    {#if !tpl.builtin}
                      <button
                        class="btn-uninstall"
                        onclick={() => handleDelete(tpl.name)}
                        aria-label="卸载 {tpl.name}"
                      >
                        <Trash2 size={14} />
                        <span>卸载</span>
                      </button>
                    {/if}
                  </div>
                {/each}
              </div>
            {/if}

          {:else if activePanel === 'tpl-search'}
            {#if tplLoading && !browseLoaded}
              <div class="panel-empty">
                <Loader size={24} class="spin" />
                <p>加载中...</p>
              </div>
            {:else if filteredAvailable.length === 0}
              <div class="panel-empty">
                <Package size={32} />
                <p>{tplSearch ? '没有匹配的模板' : '暂无可用模板'}</p>
              </div>
            {:else}
              <div class="tpl-list">
                {#each filteredAvailable as repo (repo.full_name)}
                  <div class="tpl-row">
                    <div class="tpl-info">
                      <div class="tpl-name-row">
                        <span class="tpl-name">{repo.name}</span>
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
                      <p class="tpl-desc">{repo.description}</p>
                      <span class="tpl-author">{repo.owner.login}</span>
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
      {:else}
        <div class="settings-content">
          <section id="section-general">
            <h3>通用</h3>
            <div class="setting-row">
              <div class="setting-info">
                <span class="setting-label">启用社区模板</span>
                <span class="setting-desc">允许浏览和安装第三方社区模板</span>
              </div>
              <button
                class="toggle"
                onclick={toggleCommunity}
                role="switch"
                aria-checked={communityEnabled}
                aria-label="启用社区模板"
              >
                <span class="slider" class:on={communityEnabled}></span>
              </button>
            </div>
          </section>

          <section id="section-template-dev">
            <h3>
              <BookOpen size={16} />
              模板开发
            </h3>
            <ul class="info-list">
              <li>模板协议：可执行文件，stdin 接收 Markdown，stdout 输出 Typst</li>
              <li>附带 manifest.json 描述模板元数据</li>
              <li>支持任意编程语言（Go、Rust、Python、JavaScript 等）</li>
              <li>
                <a href="https://github.com/Presto-io/template-starter" onclick={(e: MouseEvent) => { e.preventDefault(); openExternal('https://github.com/Presto-io/template-starter'); }}>
                  开发文档
                  <ExternalLink size={12} />
                </a>
              </li>
            </ul>
          </section>

          <section id="section-about">
            <h3>
              <Info size={16} />
              关于 Presto
            </h3>
            <div class="about">
              <div class="about-row">
                <span class="about-label">版本</span>
                <span class="about-value">{appVersion}</span>
              </div>
              <div class="about-row">
                <span class="about-label">更新</span>
                <span class="about-value">
                  {#if checking}
                    <RefreshCw size={12} class="spin" />
                    检查中…
                  {:else if updateInfo?.hasUpdate}
                    <a href={updateInfo.downloadURL || updateInfo.releaseURL} onclick={(e: MouseEvent) => { e.preventDefault(); openExternal(updateInfo!.downloadURL || updateInfo!.releaseURL); }} class="update-link">
                      v{updateInfo.latestVersion} 可用
                      <ExternalLink size={12} />
                    </a>
                  {:else if updateInfo && !updateInfo.hasUpdate}
                    已是最新版本
                  {:else if updateError}
                    <span class="update-error" title={updateError}>检查失败</span>
                  {:else}
                    <button class="btn-check-update" onclick={checkUpdate}>检查更新</button>
                  {/if}
                </span>
              </div>
              <div class="about-row">
                <span class="about-label">源码</span>
                <a href="https://github.com/Presto-io/Presto" onclick={(e: MouseEvent) => { e.preventDefault(); openExternal('https://github.com/Presto-io/Presto'); }} class="about-value">
                  GitHub
                  <ExternalLink size={12} />
                </a>
              </div>
              <div class="about-row">
                <span class="about-label">许可证</span>
                <span class="about-value">MIT License</span>
              </div>
            </div>
          </section>

          <section id="section-licenses">
            <h3>开源协议声明</h3>
            <p class="section-desc">Presto 基于以下开源软件构建，感谢这些项目的贡献者。</p>
            <ul class="license-list">
              <li><a href="https://go.dev" onclick={(e: MouseEvent) => { e.preventDefault(); openExternal('https://go.dev'); }} class="lib-name">Go<ExternalLink size={10} /></a><span class="lib-license">BSD-3-Clause</span></li>
              <li><a href="https://typst.app" onclick={(e: MouseEvent) => { e.preventDefault(); openExternal('https://typst.app'); }} class="lib-name">Typst<ExternalLink size={10} /></a><span class="lib-license">Apache 2.0</span></li>
              <li><a href="https://svelte.dev" onclick={(e: MouseEvent) => { e.preventDefault(); openExternal('https://svelte.dev'); }} class="lib-name">Svelte<ExternalLink size={10} /></a><span class="lib-license">MIT</span></li>
              <li><a href="https://svelte.dev/docs/kit" onclick={(e: MouseEvent) => { e.preventDefault(); openExternal('https://svelte.dev/docs/kit'); }} class="lib-name">SvelteKit<ExternalLink size={10} /></a><span class="lib-license">MIT</span></li>
              <li><a href="https://vite.dev" onclick={(e: MouseEvent) => { e.preventDefault(); openExternal('https://vite.dev'); }} class="lib-name">Vite<ExternalLink size={10} /></a><span class="lib-license">MIT</span></li>
              <li><a href="https://www.typescriptlang.org" onclick={(e: MouseEvent) => { e.preventDefault(); openExternal('https://www.typescriptlang.org'); }} class="lib-name">TypeScript<ExternalLink size={10} /></a><span class="lib-license">Apache 2.0</span></li>
              <li><a href="https://wails.io" onclick={(e: MouseEvent) => { e.preventDefault(); openExternal('https://wails.io'); }} class="lib-name">Wails<ExternalLink size={10} /></a><span class="lib-license">MIT</span></li>
              <li><a href="https://github.com/yuin/goldmark" onclick={(e: MouseEvent) => { e.preventDefault(); openExternal('https://github.com/yuin/goldmark'); }} class="lib-name">Goldmark<ExternalLink size={10} /></a><span class="lib-license">MIT</span></li>
              <li><a href="https://codemirror.net" onclick={(e: MouseEvent) => { e.preventDefault(); openExternal('https://codemirror.net'); }} class="lib-name">CodeMirror<ExternalLink size={10} /></a><span class="lib-license">MIT</span></li>
              <li><a href="https://lucide.dev" onclick={(e: MouseEvent) => { e.preventDefault(); openExternal('https://lucide.dev'); }} class="lib-name">Lucide<ExternalLink size={10} /></a><span class="lib-license">ISC</span></li>
              <li><a href="https://github.com/go-yaml/yaml" onclick={(e: MouseEvent) => { e.preventDefault(); openExternal('https://github.com/go-yaml/yaml'); }} class="lib-name">yaml.v3<ExternalLink size={10} /></a><span class="lib-license">MIT / Apache 2.0</span></li>
            </ul>
          </section>
        </div>
      {/if}
    </div>
  </div>
</div>

{#if showWarning}
  <div
    class="modal-overlay"
    onclick={cancelCommunity}
    onkeydown={(e) => { if (e.key === 'Escape') cancelCommunity(); }}
    role="dialog"
    aria-modal="true"
    aria-label="安全警告"
    tabindex="-1"
  >
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div class="modal" onclick={(e) => e.stopPropagation()} onkeydown={(e) => e.stopPropagation()}>
      <div class="modal-icon">
        <Shield size={32} />
      </div>
      <h3>安全警告</h3>
      <p>社区模板由第三方开发者提供，未经官方审核，可能存在安全风险。请仅安装你信任的模板。</p>
      <div class="modal-actions">
        <button class="btn-secondary" onclick={cancelCommunity}>取消</button>
        <button class="btn-danger" onclick={confirmCommunity}>我了解风险，启用</button>
      </div>
    </div>
  </div>
{/if}

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

  /* Two-column layout */
  .settings-layout {
    display: flex;
    gap: var(--space-xl);
    flex: 1;
    min-height: 0;
  }

  .settings-nav {
    width: 120px;
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
  .nav-item:hover {
    color: var(--color-text);
    background: var(--color-surface);
  }
  .nav-item.active {
    color: var(--color-accent);
    background: var(--color-surface);
  }

  .settings-content-wrapper {
    flex: 1;
    min-height: 0;
    max-width: 600px;
    position: relative;
  }

  .settings-content {
    height: 100%;
    overflow-y: auto;
    padding-right: var(--space-md);
  }

  /* --- Toggle (button-based, no checkbox) --- */
  .toggle {
    position: relative;
    width: 44px;
    height: 24px;
    padding: 0;
    background: none;
    border: none;
    cursor: pointer;
    flex-shrink: 0;
  }
  .slider {
    position: absolute;
    inset: 0;
    background: var(--color-surface-hover);
    border-radius: 12px;
    transition: background var(--transition);
  }
  .slider::before {
    content: '';
    position: absolute;
    width: 18px;
    height: 18px;
    left: 3px;
    bottom: 3px;
    background: white;
    border-radius: 50%;
    transition: transform var(--transition);
  }
  .slider.on { background: var(--color-accent); }
  .slider.on::before { transform: translateX(20px); }
  .toggle:focus-visible .slider {
    outline: 2px solid var(--color-accent);
    outline-offset: 2px;
  }

  section {
    margin-bottom: var(--space-xl);
    padding-bottom: var(--space-xl);
    border-bottom: 1px solid var(--color-border);
  }
  section:last-of-type { border-bottom: none; }
  h3 {
    display: flex;
    align-items: center;
    gap: var(--space-sm);
    margin: 0 0 var(--space-md);
    font-size: 0.9375rem;
    color: var(--color-text);
  }
  .setting-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: var(--space-md);
    background: var(--color-surface);
    border-radius: var(--radius-md);
  }
  .setting-info { display: flex; flex-direction: column; gap: var(--space-xs); }
  .setting-label { font-size: 0.875rem; font-weight: 500; }
  .setting-desc { font-size: 0.75rem; color: var(--color-muted); }
  .info-list {
    list-style: none;
    padding: 0;
    margin: 0;
    display: flex;
    flex-direction: column;
    gap: var(--space-sm);
    font-size: 0.8125rem;
    color: var(--color-muted);
  }
  .info-list a {
    display: inline-flex;
    align-items: center;
    gap: var(--space-xs);
    color: var(--color-accent);
    text-decoration: none;
    cursor: pointer;
    transition: opacity var(--transition);
  }
  .info-list a:hover { opacity: 0.8; }
  .about {
    display: flex;
    flex-direction: column;
    gap: var(--space-sm);
  }
  .about-row {
    display: flex;
    justify-content: space-between;
    padding: var(--space-sm) var(--space-md);
    background: var(--color-surface);
    border-radius: var(--radius-sm);
    font-size: 0.8125rem;
  }
  .about-label { color: var(--color-muted); }
  .about-value {
    display: inline-flex;
    align-items: center;
    gap: var(--space-xs);
    font-family: var(--font-mono);
    font-size: 0.8125rem;
  }
  a.about-value {
    color: var(--color-accent);
    text-decoration: none;
    cursor: pointer;
    transition: opacity var(--transition);
  }
  a.about-value:hover { opacity: 0.8; }
  .btn-check-update {
    background: none;
    border: 1px solid var(--color-border);
    border-radius: var(--radius-sm);
    color: var(--color-accent);
    font-size: 0.75rem;
    padding: 2px 8px;
    cursor: pointer;
    transition: all var(--transition);
  }
  .btn-check-update:hover { background: var(--color-surface-hover); }
  .update-link {
    display: inline-flex;
    align-items: center;
    gap: var(--space-xs);
    color: var(--color-accent);
    text-decoration: none;
    cursor: pointer;
    font-weight: 500;
  }
  .update-link:hover { opacity: 0.8; }
  .update-error {
    color: var(--color-danger);
    font-size: 0.75rem;
  }
  :global(.spin) {
    animation: spin 1s linear infinite;
  }
  @keyframes spin {
    from { transform: rotate(0deg); }
    to { transform: rotate(360deg); }
  }
  .section-desc {
    font-size: 0.8125rem;
    color: var(--color-muted);
    margin: 0 0 var(--space-md);
  }
  .license-list {
    list-style: none;
    padding: 0;
    margin: 0;
    display: flex;
    flex-direction: column;
    gap: var(--space-xs);
  }
  .license-list li {
    display: flex;
    justify-content: space-between;
    padding: var(--space-sm) var(--space-md);
    background: var(--color-surface);
    border-radius: var(--radius-sm);
    font-size: 0.8125rem;
  }
  .lib-name {
    font-weight: 500;
    display: inline-flex;
    align-items: center;
    gap: 4px;
    color: var(--color-text);
    text-decoration: none;
    cursor: pointer;
    transition: color var(--transition);
  }
  a.lib-name:hover { color: var(--color-accent); }
  a.lib-name :global(svg) {
    opacity: 0;
    transition: opacity var(--transition);
  }
  a.lib-name:hover :global(svg) { opacity: 1; }
  .lib-license { color: var(--color-muted); font-family: var(--font-mono); }

  /* --- Panel overlay (covers settings content) --- */
  .panel-overlay {
    position: absolute;
    inset: 0;
    background: var(--color-bg);
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }
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
  .tpl-list {
    display: flex;
    flex-direction: column;
    gap: var(--space-sm);
    overflow-y: auto;
    flex: 1;
    padding-right: var(--space-md);
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
  .badge-builtin {
    font-size: 0.625rem;
    font-weight: 600;
    padding: 1px 6px;
    border-radius: 4px;
    background: var(--color-accent);
    color: var(--color-bg);
    letter-spacing: 0.02em;
  }
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

  /* --- Modal --- */
  .modal-overlay {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.6);
    backdrop-filter: blur(4px);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 50;
  }
  .modal {
    background: var(--color-bg-elevated);
    padding: var(--space-xl);
    border-radius: var(--radius-lg);
    max-width: 420px;
    width: 90%;
    border: 1px solid var(--color-border);
  }
  .modal-icon {
    color: var(--color-danger);
    margin-bottom: var(--space-md);
  }
  .modal h3 {
    margin: 0 0 var(--space-sm);
    font-size: 1rem;
  }
  .modal p {
    font-size: 0.8125rem;
    color: var(--color-muted);
    line-height: 1.6;
    margin: 0 0 var(--space-lg);
  }
  .modal-actions {
    display: flex;
    gap: var(--space-sm);
    justify-content: flex-end;
  }
  .btn-secondary, .btn-danger {
    padding: var(--space-sm) var(--space-md);
    border-radius: var(--radius-md);
    font-size: 0.8125rem;
    font-weight: 500;
    cursor: pointer;
    transition: all var(--transition);
    border: none;
  }
  .btn-secondary {
    background: var(--color-secondary);
    color: var(--color-text);
  }
  .btn-secondary:hover { background: var(--color-surface-hover); }
  .btn-danger {
    background: var(--color-danger);
    color: white;
  }
  .btn-danger:hover { opacity: 0.9; }
</style>
