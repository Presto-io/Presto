<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { fly, fade } from 'svelte/transition';
  import { cubicOut } from 'svelte/easing';
  import { ExternalLink, Shield, Info, BookOpen, ArrowLeft, RefreshCw, Search, Package, ShoppingBag, Trash2, Loader, Upload, Pencil, Check, X, AlertTriangle, Settings, Scale, HelpCircle } from 'lucide-svelte';
  import { goto } from '$app/navigation';
  import { listTemplates, deleteTemplate, importTemplateZip, renameTemplate } from '$lib/api/client';
  import type { Template } from '$lib/api/types';
  import { templateStore } from '$lib/stores/templates.svelte';
  import { triggerAction, resetWizard } from '$lib/stores/wizard.svelte';

  const isMac = typeof navigator !== 'undefined' && /Mac|iPhone|iPad/.test(navigator.userAgent);
  const mod = isMac ? '⌘' : 'Ctrl';

  // --- Settings state ---
  let communityEnabled = $state(false);
  let showWarning = $state(false);
  let appVersion = $state(__APP_VERSION__);
  let updateInfo = $state<{ hasUpdate: boolean; latestVersion: string; downloadURL: string; releaseURL: string } | null>(null);
  let checking = $state(false);
  let updateError = $state('');
  let activeSection = $state('general');
  let activePanel = $state<'tpl-manage' | null>(null);

  // --- Template state (migrated from /templates) ---
  let installed: Template[] = $state([]);
  let tplLoading = $state(false);
  let tplSearch = $state('');
  let installedLoaded = $state(false);
  let selectedKeywords: string[] = $state([]);

  let allKeywords = $derived(
    [...new Set(installed.flatMap(tpl => tpl.keywords ?? []))].sort()
  );

  let filteredInstalled = $derived(
    installed.filter(tpl => {
      const q = tplSearch.toLowerCase();
      const matchesSearch = !q ||
        (tpl.displayName || tpl.name).toLowerCase().includes(q) ||
        tpl.description.toLowerCase().includes(q) ||
        (tpl.keywords ?? []).some(k => k.toLowerCase().includes(q));
      const matchesKeywords = selectedKeywords.length === 0 ||
        selectedKeywords.every(k => (tpl.keywords ?? []).includes(k));
      return matchesSearch && matchesKeywords;
    })
  );

  import type { Component } from 'svelte';

  const sections: { id: string; label: string; icon: Component }[] = [
    { id: 'general', label: '通用', icon: Settings },
    { id: 'help', label: '帮助', icon: HelpCircle },
    { id: 'template-dev', label: '模板开发', icon: BookOpen },
    { id: 'about', label: '关于', icon: Info },
    { id: 'licenses', label: '开源协议', icon: Scale },
  ];

  let panelTabs = $derived(communityEnabled ? [
    { id: 'tpl-manage' as const, label: '模板管理', icon: Package },
    { id: 'tpl-store' as const, label: '模板商店', icon: ShoppingBag },
  ] : []);

  declare global {
    interface Window {
      go?: { main: { App: {
        GetVersion: () => Promise<string>;
        CheckForUpdate: () => Promise<{ hasUpdate: boolean; currentVersion: string; latestVersion: string; downloadURL: string; releaseURL: string }>;
        DeleteTemplate: (name: string) => Promise<void>;
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
    window.addEventListener('templates-changed', onTemplatesChanged);
  });

  onDestroy(() => {
    window.removeEventListener('templates-changed', onTemplatesChanged);
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

  function togglePanel(id: 'tpl-manage' | 'tpl-store') {
    if (id === 'tpl-store') {
      goto('/store');
      return;
    }
    if (activePanel === id) {
      activePanel = null;
      tplSearch = '';
      selectedKeywords = [];
    } else {
      activePanel = id;
      activeSection = '';
      tplSearch = '';
      selectedKeywords = [];
      if (id === 'tpl-manage' && !installedLoaded) loadInstalled();
    }
  }

  function toggleKeyword(keyword: string) {
    if (selectedKeywords.includes(keyword)) {
      selectedKeywords = selectedKeywords.filter(k => k !== keyword);
    } else {
      selectedKeywords = [...selectedKeywords, keyword];
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
    // Wizard: notify that community templates are now enabled
    setTimeout(() => triggerAction('community-template-toggle'), 500);
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

  let deleteConfirm = $state<string | null>(null);

  function handleDelete(name: string) {
    deleteConfirm = name;
  }

  async function confirmDelete() {
    const name = deleteConfirm;
    if (!name) return;
    deleteConfirm = null;
    try {
      if (window.go?.main?.App?.DeleteTemplate) {
        await window.go.main.App.DeleteTemplate(name);
      } else {
        await deleteTemplate(name);
      }
      installed = (await listTemplates()) ?? [];
      await templateStore.refresh();
      showImportToast(`模板 "${name}" 已卸载`, 'success');
    } catch (err) {
      showImportToast(err instanceof Error ? err.message : String(err), 'error');
    }
  }

  // --- ZIP import ---
  let importingZip = $state(false);
  let zipInput: HTMLInputElement | undefined = $state();
  let conflictModal = $state<{ file: File; conflicts: string[] } | null>(null);
  let importToast = $state<{ message: string; type: 'success' | 'error' } | null>(null);
  let importToastTimer: ReturnType<typeof setTimeout>;

  function showImportToast(message: string, type: 'success' | 'error') {
    clearTimeout(importToastTimer);
    importToast = { message, type };
    importToastTimer = setTimeout(() => { importToast = null; }, 2500);
  }

  function handleImportZipClick() {
    zipInput?.click();
  }

  async function handleZipFileSelected(e: Event) {
    const input = e.target as HTMLInputElement;
    const file = input.files?.[0];
    if (!file) return;
    input.value = '';
    importingZip = true;
    try {
      const tpls = await importTemplateZip(file);
      installed = (await listTemplates()) ?? [];
      installedLoaded = true;
      const names = tpls.map(t => t.displayName || t.name).join('、');
      showImportToast(`模板 "${names}" 导入成功`, 'success');
    } catch (err: any) {
      if (err.conflicts) {
        conflictModal = { file, conflicts: err.conflicts };
      } else {
        showImportToast(err instanceof Error ? err.message : String(err), 'error');
      }
    } finally {
      importingZip = false;
    }
  }

  async function handleConflictResolve(strategy: 'overwrite' | 'skip' | 'rename') {
    if (!conflictModal) return;
    const { file } = conflictModal;
    conflictModal = null;
    importingZip = true;
    try {
      const tpls = await importTemplateZip(file, strategy);
      installed = (await listTemplates()) ?? [];
      installedLoaded = true;
      const names = tpls.map(t => t.displayName || t.name).join('、');
      const suffix = strategy === 'rename' ? '（已自动重命名）' : strategy === 'skip' ? '（已跳过重复）' : '（已覆盖）';
      showImportToast(`模板 "${names}" 导入成功${suffix}`, 'success');
    } catch (err) {
      showImportToast(err instanceof Error ? err.message : String(err), 'error');
    } finally {
      importingZip = false;
    }
  }

  // --- Rename (displayName) ---
  let renamingTpl = $state('');
  let renameInput = $state('');

  function startRename(tpl: Template) {
    renamingTpl = tpl.name;
    renameInput = tpl.displayName || tpl.name;
  }

  function cancelRename() {
    renamingTpl = '';
    renameInput = '';
  }

  async function confirmRename() {
    const tplName = renamingTpl;
    const newDisplayName = renameInput.trim();
    if (!newDisplayName) {
      cancelRename();
      return;
    }
    // Find current displayName to check if actually changed
    const current = installed.find(t => t.name === tplName);
    if (current && newDisplayName === (current.displayName || current.name)) {
      cancelRename();
      return;
    }
    try {
      await renameTemplate(tplName, newDisplayName);
      installed = (await listTemplates()) ?? [];
      installedLoaded = true;
      showImportToast(`模板已重命名为 "${newDisplayName}"`, 'success');
    } catch (err) {
      showImportToast(err instanceof Error ? err.message : String(err), 'error');
    } finally {
      cancelRename();
    }
  }

  // --- Listen for external template changes (drag-drop import from layout) ---
  function onTemplatesChanged() {
    loadInstalled();
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
        {@const Icon = sec.icon}
        <button
          class="nav-item"
          class:active={!activePanel && activeSection === sec.id}
          onclick={() => scrollTo(sec.id)}
        >
          <Icon size={14} />
          {sec.label}
        </button>
      {/each}
      {#if panelTabs.length > 0}
        <div class="nav-divider" transition:fade={{ duration: 200 }}></div>
        {#each panelTabs as tab (tab.id)}
          {@const Icon = tab.icon}
          <button
            class="nav-item"
            class:active={activePanel === tab.id}
            onclick={() => togglePanel(tab.id)}
            transition:fly={{ x: -12, duration: 300, easing: cubicOut }}
          >
            <Icon size={14} />
            {tab.label}
          </button>
        {/each}
      {/if}
    </nav>

    <div class="settings-content-wrapper">
      {#if activePanel}
        <div class="panel-overlay">
          {#if activePanel === 'tpl-manage'}
            <div class="panel-header">
              <div class="panel-search">
                <Search size={14} />
                <input
                  type="text"
                  placeholder="搜索已安装模板…"
                  bind:value={tplSearch}
                />
              </div>
              <button
                class="btn-import"
                onclick={handleImportZipClick}
                disabled={importingZip}
                title="从 ZIP 文件导入模板"
              >
                {#if importingZip}
                  <Loader size={14} class="spin" />
                {:else}
                  <Upload size={14} />
                {/if}
                <span>从 ZIP 导入</span>
              </button>
              <input
                bind:this={zipInput}
                type="file"
                accept=".zip"
                onchange={handleZipFileSelected}
                hidden
              />
            </div>
          {/if}

          {#if activePanel === 'tpl-manage'}
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
                        {#if renamingTpl === tpl.name}
                          <input
                            class="rename-input"
                            type="text"
                            bind:value={renameInput}
                            onkeydown={(e) => { if (e.key === 'Enter') confirmRename(); if (e.key === 'Escape') cancelRename(); }}
                          />
                          <button class="btn-rename-action confirm" onclick={confirmRename} aria-label="确认重命名">
                            <Check size={12} />
                          </button>
                          <button class="btn-rename-action cancel" onclick={cancelRename} aria-label="取消重命名">
                            <X size={12} />
                          </button>
                        {:else}
                          <span class="tpl-name">{tpl.displayName || tpl.name}</span>
                          <span class="tpl-version">v{tpl.version}</span>
                          {#if tpl.builtin}
                            <span class="badge-builtin">内置</span>
                          {/if}
                        {/if}
                      </div>
                      <p class="tpl-desc">{tpl.description}</p>
                      {#if tpl.keywords && tpl.keywords.length > 0}
                        <div class="tpl-keywords">
                          {#each tpl.keywords as kw (kw)}
                            <span class="keyword-badge">{kw}</span>
                          {/each}
                        </div>
                      {/if}
                      <span class="tpl-author">{tpl.author}</span>
                    </div>
                    {#if !tpl.builtin}
                      <div class="tpl-actions">
                        {#if renamingTpl !== tpl.name}
                          <button
                            class="btn-rename"
                            onclick={() => startRename(tpl)}
                            aria-label="重命名 {tpl.displayName || tpl.name}"
                          >
                            <Pencil size={12} />
                          </button>
                        {/if}
                        <button
                          class="btn-uninstall"
                          onclick={() => handleDelete(tpl.name)}
                          aria-label="卸载 {tpl.name}"
                        >
                          <Trash2 size={14} />
                          <span>卸载</span>
                        </button>
                      </div>
                    {/if}
                  </div>
                {/each}
              </div>
            {/if}
          {/if}
        </div>
      {:else}
        <div class="settings-content">
          <section id="section-general">
            <h3><Settings size={16} /> 通用</h3>
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

          <section id="section-help">
            <h3><HelpCircle size={16} /> 帮助</h3>

            <h4 class="subsection-title">功能</h4>
            <div class="feature-list">
              <div class="feature-row">
                <span class="feature-name">Markdown 编辑</span>
                <span class="feature-desc">支持实时预览的 Markdown 编辑器</span>
              </div>
              <div class="feature-row">
                <span class="feature-name">PDF 导出</span>
                <span class="feature-desc">将 Markdown 转换为排版精美的 PDF</span>
              </div>
              <div class="feature-row">
                <span class="feature-name">模板系统</span>
                <span class="feature-desc">切换不同文档模板，支持第三方模板</span>
              </div>
              <div class="feature-row">
                <span class="feature-name">批量转换</span>
                <span class="feature-desc">一次转换多个 Markdown 文件为 PDF</span>
              </div>
              <div class="feature-row">
                <span class="feature-name">拖放导入</span>
                <span class="feature-desc">拖拽文件到窗口即可打开或批量导入</span>
              </div>
              <div class="feature-row">
                <span class="feature-name">社区模板</span>
                <span class="feature-desc">浏览和安装第三方社区模板</span>
              </div>
            </div>

            <h4 class="subsection-title">快捷键</h4>
            <div class="shortcut-list">
              <div class="shortcut-row">
                <span class="shortcut-action">打开文件</span>
                <span class="shortcut-keys"><kbd>{mod}</kbd><kbd>O</kbd></span>
              </div>
              <div class="shortcut-row">
                <span class="shortcut-action">导出 PDF</span>
                <span class="shortcut-keys"><kbd>{mod}</kbd><kbd>E</kbd></span>
              </div>
              <div class="shortcut-row">
                <span class="shortcut-action">打开设置</span>
                <span class="shortcut-keys"><kbd>{mod}</kbd><kbd>,</kbd></span>
              </div>
              <div class="shortcut-row">
                <span class="shortcut-action">模板管理</span>
                <span class="shortcut-keys"><kbd>{mod}</kbd><kbd>⇧</kbd><kbd>T</kbd></span>
              </div>
              <div class="shortcut-row">
                <span class="shortcut-action">搜索 / 替换</span>
                <span class="shortcut-keys"><kbd>{mod}</kbd><kbd>F</kbd></span>
              </div>
              <div class="shortcut-row">
                <span class="shortcut-action">撤销</span>
                <span class="shortcut-keys"><kbd>{mod}</kbd><kbd>Z</kbd></span>
              </div>
              <div class="shortcut-row">
                <span class="shortcut-action">重做</span>
                <span class="shortcut-keys"><kbd>{mod}</kbd><kbd>⇧</kbd><kbd>Z</kbd></span>
              </div>
            </div>

            <div class="setting-row" style="margin-top: var(--space-lg)">
              <div class="setting-info">
                <span class="setting-label">重置引导</span>
                <span class="setting-desc">重置所有引导提示，重新展示操作引导</span>
              </div>
              <button class="btn-reset-wizard" onclick={resetWizard}>重置引导</button>
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
            <h3><Scale size={16} /> 开源协议声明</h3>
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

{#if conflictModal}
  <div
    class="modal-overlay"
    onclick={() => conflictModal = null}
    onkeydown={(e) => { if (e.key === 'Escape') conflictModal = null; }}
    role="dialog"
    aria-modal="true"
    aria-label="模板名称冲突"
    tabindex="-1"
  >
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div class="modal" onclick={(e) => e.stopPropagation()} onkeydown={(e) => e.stopPropagation()}>
      <div class="modal-icon conflict">
        <AlertTriangle size={32} />
      </div>
      <h3>模板名称冲突</h3>
      <p>以下模板已存在：{conflictModal.conflicts.join('、')}。请选择处理方式：</p>
      <div class="modal-actions conflict-actions">
        <button class="btn-secondary" onclick={() => conflictModal = null}>取消</button>
        <button class="btn-secondary" onclick={() => handleConflictResolve('skip')}>跳过</button>
        <button class="btn-secondary" onclick={() => handleConflictResolve('rename')}>自动重命名</button>
        <button class="btn-danger" onclick={() => handleConflictResolve('overwrite')}>覆盖</button>
      </div>
    </div>
  </div>
{/if}

{#if deleteConfirm}
  <div
    class="modal-overlay"
    onclick={() => deleteConfirm = null}
    onkeydown={(e) => { if (e.key === 'Escape') deleteConfirm = null; }}
    role="dialog"
    aria-modal="true"
    aria-label="确认卸载"
    tabindex="-1"
  >
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div class="modal" onclick={(e) => e.stopPropagation()} onkeydown={(e) => e.stopPropagation()}>
      <div class="modal-icon">
        <Trash2 size={32} />
      </div>
      <h3>确认卸载</h3>
      <p>确定卸载模板 "{deleteConfirm}"？此操作不可撤销。</p>
      <div class="modal-actions">
        <button class="btn-secondary" onclick={() => deleteConfirm = null}>取消</button>
        <button class="btn-danger" onclick={confirmDelete}>卸载</button>
      </div>
    </div>
  </div>
{/if}

{#if importToast}
  <div class="import-toast" class:toast-error={importToast.type === 'error'}>
    {importToast.message}
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
    width: 180px;
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
    background: var(--color-toggle-knob);
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
  .btn-reset-wizard {
    background: none;
    border: 1px solid var(--color-border);
    border-radius: var(--radius-sm);
    color: var(--color-muted);
    font-size: 0.75rem;
    padding: 2px 8px;
    cursor: pointer;
    transition: all var(--transition);
  }
  .btn-reset-wizard:hover { color: var(--color-accent); border-color: var(--color-accent); }
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
  .subsection-title {
    font-size: 0.8125rem;
    font-weight: 600;
    color: var(--color-muted);
    margin: 0 0 var(--space-sm);
  }
  .feature-list {
    display: flex;
    flex-direction: column;
    gap: var(--space-xs);
    margin-bottom: var(--space-lg);
  }
  .feature-row {
    display: flex;
    gap: var(--space-md);
    padding: var(--space-sm) var(--space-md);
    background: var(--color-surface);
    border-radius: var(--radius-sm);
    font-size: 0.8125rem;
  }
  .feature-name {
    font-weight: 500;
    white-space: nowrap;
    min-width: 80px;
  }
  .feature-desc {
    color: var(--color-muted);
  }
  .shortcut-list {
    display: flex;
    flex-direction: column;
    gap: var(--space-xs);
  }
  .shortcut-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: var(--space-sm) var(--space-md);
    background: var(--color-surface);
    border-radius: var(--radius-sm);
    font-size: 0.8125rem;
  }
  .shortcut-action {
    color: var(--color-text);
  }
  .shortcut-keys {
    display: inline-flex;
    align-items: center;
    gap: 3px;
  }
  .shortcut-keys kbd {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    min-width: 22px;
    height: 22px;
    padding: 0 5px;
    background: var(--color-bg);
    border: 1px solid var(--color-border);
    border-radius: 4px;
    font-family: var(--font-ui);
    font-size: 0.6875rem;
    color: var(--color-muted);
    box-shadow: 0 1px 0 var(--color-border);
    line-height: 1;
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
  .btn-import:disabled { opacity: 0.5; cursor: not-allowed; }
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
  .keyword-chip:hover {
    border-color: var(--color-accent);
    color: var(--color-text);
  }
  .keyword-chip.active {
    background: var(--color-accent);
    color: var(--color-bg);
    border-color: var(--color-accent);
  }
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
  .btn-uninstall:hover {
    background: var(--color-danger);
    color: var(--color-on-danger);
  }

  /* --- Modal --- */
  .modal-overlay {
    position: fixed;
    inset: 0;
    background: var(--color-backdrop);
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
    color: var(--color-on-danger);
  }
  .btn-danger:hover { opacity: 0.9; }

  /* --- Rename --- */
  .tpl-actions {
    display: flex;
    align-items: center;
    gap: var(--space-xs);
    flex-shrink: 0;
  }
  .rename-input {
    flex: 1;
    min-width: 0;
    padding: 2px var(--space-sm);
    background: var(--color-bg);
    border: 1px solid var(--color-accent);
    border-radius: var(--radius-sm);
    color: var(--color-text);
    font-size: 0.875rem;
    font-family: var(--font-mono);
    outline: none;
  }
  .btn-rename-action {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 20px;
    height: 20px;
    border: none;
    border-radius: var(--radius-sm);
    cursor: pointer;
    transition: all var(--transition);
    flex-shrink: 0;
  }
  .btn-rename-action.confirm {
    background: var(--color-accent);
    color: var(--color-bg);
  }
  .btn-rename-action.confirm:hover { opacity: 0.85; }
  .btn-rename-action.cancel {
    background: var(--color-surface);
    color: var(--color-muted);
  }
  .btn-rename-action.cancel:hover { color: var(--color-danger); }
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
  .btn-rename:hover {
    border-color: var(--color-accent);
    color: var(--color-accent);
  }

  /* --- Conflict modal --- */
  .modal-icon.conflict { color: var(--color-warning); }
  .conflict-actions { flex-wrap: wrap; }

  /* --- Import toast --- */
  .import-toast {
    position: fixed;
    bottom: var(--space-xl);
    left: 50%;
    transform: translateX(-50%);
    z-index: 9001;
    padding: var(--space-sm) var(--space-lg);
    background: var(--color-success);
    color: var(--color-bg);
    border-radius: var(--radius-md);
    font-size: 0.8125rem;
    font-weight: 500;
    pointer-events: none;
    animation: toast-in 200ms ease-out;
  }
  .import-toast.toast-error { background: var(--color-danger); }
  @keyframes toast-in {
    from { opacity: 0; transform: translateX(-50%) translateY(8px); }
    to { opacity: 1; transform: translateX(-50%) translateY(0); }
  }
</style>
