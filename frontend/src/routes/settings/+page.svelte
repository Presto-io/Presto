<script lang="ts">
  import { onMount } from 'svelte';
  import { ExternalLink, Shield, Info, BookOpen, ArrowLeft } from 'lucide-svelte';
  import { goto } from '$app/navigation';

  let communityEnabled = $state(false);
  let showWarning = $state(false);

  onMount(() => {
    communityEnabled = localStorage.getItem('communityTemplates') === 'true';
  });

  function toggleCommunity() {
    if (!communityEnabled) {
      showWarning = true;
    } else {
      communityEnabled = false;
      localStorage.setItem('communityTemplates', 'false');
    }
  }

  function confirmCommunity() {
    communityEnabled = true;
    showWarning = false;
    localStorage.setItem('communityTemplates', 'true');
  }
</script>

<div class="page">
  <div class="page-header">
    <button class="btn-back" onclick={() => goto('/')} aria-label="返回编辑器">
      <ArrowLeft size={16} />
    </button>
    <h2>设置</h2>
  </div>

  <section>
    <h3>通用</h3>
    <div class="setting-row">
      <div class="setting-info">
        <span class="setting-label">启用社区模板</span>
        <span class="setting-desc">允许浏览和安装第三方社区模板</span>
      </div>
      <label class="toggle">
        <input type="checkbox" checked={communityEnabled} onchange={toggleCommunity} />
        <span class="slider"></span>
      </label>
    </div>
  </section>

  <section>
    <h3>
      <BookOpen size={16} />
      模板开发
    </h3>
    <ul class="info-list">
      <li>模板协议：可执行文件，stdin 接收 Markdown，stdout 输出 Typst</li>
      <li>附带 manifest.json 描述模板元数据</li>
      <li>支持任意编程语言（Go、Rust、Python、JavaScript 等）</li>
      <li>
        <a href="https://github.com/Presto-io/template-starter" target="_blank" rel="noopener noreferrer">
          开发文档
          <ExternalLink size={12} />
        </a>
      </li>
    </ul>
  </section>

  <section>
    <h3>
      <Info size={16} />
      关于 Presto
    </h3>
    <div class="about">
      <div class="about-row">
        <span class="about-label">版本</span>
        <span class="about-value">0.1.0</span>
      </div>
      <div class="about-row">
        <span class="about-label">源码</span>
        <a href="https://github.com/Presto-io/Presto" target="_blank" rel="noopener noreferrer" class="about-value">
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

  <section>
    <h3>开源协议声明</h3>
    <p class="section-desc">Presto 基于以下开源软件构建，感谢这些项目的贡献者。</p>
    <ul class="license-list">
      <li><span class="lib-name">Wails</span><span class="lib-license">MIT</span></li>
      <li><span class="lib-name">Typst</span><span class="lib-license">Apache 2.0</span></li>
      <li><span class="lib-name">typst.ts</span><span class="lib-license">Apache 2.0</span></li>
      <li><span class="lib-name">Goldmark</span><span class="lib-license">MIT</span></li>
      <li><span class="lib-name">CodeMirror</span><span class="lib-license">MIT</span></li>
      <li><span class="lib-name">Svelte</span><span class="lib-license">MIT</span></li>
      <li><span class="lib-name">Go</span><span class="lib-license">BSD-3-Clause</span></li>
    </ul>
  </section>
</div>

{#if showWarning}
  <div
    class="modal-overlay"
    onclick={() => showWarning = false}
    onkeydown={(e) => { if (e.key === 'Escape') showWarning = false; }}
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
        <button class="btn-secondary" onclick={() => showWarning = false}>取消</button>
        <button class="btn-danger" onclick={confirmCommunity}>我了解风险，启用</button>
      </div>
    </div>
  </div>
{/if}

<style>
  .page {
    padding: var(--space-xl);
    padding-top: 48px;
    max-width: 700px;
    margin: 0 auto;
    overflow-y: auto;
    height: 100%;
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
  .toggle {
    position: relative;
    width: 44px;
    height: 24px;
    cursor: pointer;
  }
  .toggle input { opacity: 0; width: 0; height: 0; position: absolute; }
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
  .toggle input:checked + .slider { background: var(--color-accent); }
  .toggle input:checked + .slider::before { transform: translateX(20px); }
  .toggle input:focus-visible + .slider {
    outline: 2px solid var(--color-accent);
    outline-offset: 2px;
  }
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
  }
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
  .lib-name { font-weight: 500; }
  .lib-license { color: var(--color-muted); font-family: var(--font-mono); }
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
