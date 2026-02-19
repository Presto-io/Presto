<script lang="ts">
  import TemplateSelector from '$lib/components/TemplateSelector.svelte';
  import { convertAndCompile } from '$lib/api/client';
  import { ArrowLeft, Upload, FileText, Download, X, Loader, CheckCircle, AlertCircle } from 'lucide-svelte';
  import { goto } from '$app/navigation';

  let selectedTemplate = $state('');
  let files: File[] = $state([]);
  let results: { name: string; blob?: Blob; error?: string }[] = $state([]);
  let processing = $state(false);
  let dragOver = $state(false);
  let fileInput: HTMLInputElement | undefined = $state();

  let successCount = $derived(results.filter(r => r.blob).length);
  let errorCount = $derived(results.filter(r => r.error).length);

  function handleDrop(e: DragEvent) {
    e.preventDefault();
    dragOver = false;
    const dropped = Array.from(e.dataTransfer?.files ?? []).filter(f =>
      f.name.endsWith('.md') || f.name.endsWith('.markdown') || f.name.endsWith('.txt')
    );
    files = [...files, ...dropped];
  }

  function handleFileInput(e: Event) {
    const input = e.target as HTMLInputElement;
    files = [...files, ...Array.from(input.files ?? [])];
    input.value = '';
  }

  function removeFile(index: number) {
    files = files.filter((_, i) => i !== index);
  }

  function clearAll() {
    files = [];
    results = [];
  }

  async function convertAll() {
    if (!selectedTemplate || files.length === 0) return;
    processing = true;
    results = [];

    for (const file of files) {
      try {
        const text = await file.text();
        const blob = await convertAndCompile(text, selectedTemplate);
        results = [...results, { name: file.name.replace(/\.\w+$/, '.pdf'), blob }];
      } catch (e) {
        results = [...results, { name: file.name, error: String(e) }];
      }
    }
    processing = false;
  }

  function downloadOne(r: { name: string; blob?: Blob }) {
    if (!r.blob) return;
    const url = URL.createObjectURL(r.blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = r.name;
    a.click();
    URL.revokeObjectURL(url);
  }

  function downloadAll() {
    for (const r of results) {
      if (r.blob) downloadOne(r);
    }
  }
</script>

<div class="page">
  <div class="page-header">
    <button class="btn-back" onclick={() => goto('/')} aria-label="返回编辑器">
      <ArrowLeft size={16} />
    </button>
    <h2>批量转换</h2>
  </div>

  <div class="batch-content">
    <!-- Controls -->
    <div class="control-bar">
      <div class="control-left">
        <TemplateSelector bind:selected={selectedTemplate} />
      </div>
      <div class="control-right">
        <button class="btn-action" onclick={() => fileInput?.click()}>
          <Upload size={14} />
          <span>选择文件</span>
        </button>
        <input
          bind:this={fileInput}
          type="file"
          accept=".md,.markdown,.txt"
          multiple
          onchange={handleFileInput}
          hidden
        />
        {#if files.length > 0}
          <button class="btn-action subtle" onclick={clearAll}>
            <X size={14} />
            <span>清空</span>
          </button>
        {/if}
      </div>
    </div>

    <!-- Drop zone / empty state -->
    {#if files.length === 0 && results.length === 0}
      <div
        class="drop-zone"
        class:drag-over={dragOver}
        ondrop={handleDrop}
        ondragover={(e) => { e.preventDefault(); dragOver = true; }}
        ondragleave={() => dragOver = false}
        role="region"
        aria-label="拖拽文件区域"
      >
        <Upload size={28} strokeWidth={1.5} />
        <p class="drop-title">拖拽 Markdown 文件到此处</p>
        <p class="drop-hint">支持 .md .markdown .txt 格式，可同时添加多个文件</p>
      </div>
    {:else}
      <!-- Compact drop target when files exist -->
      <div
        class="drop-zone compact"
        class:drag-over={dragOver}
        ondrop={handleDrop}
        ondragover={(e) => { e.preventDefault(); dragOver = true; }}
        ondragleave={() => dragOver = false}
        role="region"
        aria-label="拖拽更多文件"
      >
        <Upload size={14} />
        <span>拖拽更多文件到此处</span>
      </div>
    {/if}

    <!-- File list -->
    {#if files.length > 0}
      <div class="section">
        <div class="section-header">
          <h3>待转换文件</h3>
          <span class="section-count">{files.length}</span>
        </div>
        <div class="file-list">
          {#each files as file, i (file.name + i)}
            <div class="file-row">
              <FileText size={14} />
              <span class="file-name">{file.name}</span>
              <span class="file-size">{(file.size / 1024).toFixed(1)} KB</span>
              <button class="btn-icon" onclick={() => removeFile(i)} aria-label="移除 {file.name}">
                <X size={12} />
              </button>
            </div>
          {/each}
        </div>
      </div>

      <!-- Convert button -->
      <button
        class="btn-convert"
        onclick={convertAll}
        disabled={processing || !selectedTemplate}
      >
        {#if processing}
          <Loader size={14} class="spin" />
          <span>转换中…</span>
        {:else}
          <span>转换全部 ({files.length} 个文件)</span>
        {/if}
      </button>
    {/if}

    <!-- Results -->
    {#if results.length > 0}
      <div class="section">
        <div class="section-header">
          <h3>转换结果</h3>
          <div class="result-summary">
            {#if successCount > 0}
              <span class="badge success"><CheckCircle size={10} /> {successCount} 成功</span>
            {/if}
            {#if errorCount > 0}
              <span class="badge error"><AlertCircle size={10} /> {errorCount} 失败</span>
            {/if}
          </div>
        </div>
        <div class="file-list">
          {#each results as r (r.name)}
            <div class="file-row" class:has-error={!!r.error}>
              {#if r.blob}
                <CheckCircle size={14} />
              {:else}
                <AlertCircle size={14} />
              {/if}
              <span class="file-name">{r.name}</span>
              {#if r.blob}
                <button class="btn-dl" onclick={() => downloadOne(r)}>
                  <Download size={12} />
                  <span>下载</span>
                </button>
              {:else}
                <span class="error-text" title={r.error}>{r.error}</span>
              {/if}
            </div>
          {/each}
        </div>
        {#if successCount > 1}
          <button class="btn-action" onclick={downloadAll} style="margin-top: var(--space-sm); align-self: flex-end;">
            <Download size={14} />
            <span>全部下载</span>
          </button>
        {/if}
      </div>
    {/if}
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

  .batch-content {
    flex: 1;
    min-height: 0;
    max-width: 640px;
    display: flex;
    flex-direction: column;
    overflow-y: auto;
    gap: var(--space-lg);
  }

  /* Control bar */
  .control-bar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-md);
    flex-shrink: 0;
  }
  .control-left {
    display: flex;
    align-items: center;
    gap: var(--space-sm);
  }
  .control-right {
    display: flex;
    align-items: center;
    gap: var(--space-xs);
  }

  .btn-action {
    display: inline-flex;
    align-items: center;
    gap: var(--space-xs);
    height: 28px;
    padding: 0 var(--space-md);
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-sm);
    color: var(--color-text);
    font-size: 0.75rem;
    cursor: pointer;
    transition: all var(--transition);
    white-space: nowrap;
  }
  .btn-action:hover { border-color: var(--color-accent); color: var(--color-accent); }
  .btn-action.subtle { color: var(--color-muted); }
  .btn-action.subtle:hover { border-color: var(--color-danger); color: var(--color-danger); }

  /* Drop zone */
  .drop-zone {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: var(--space-sm);
    padding: var(--space-2xl) var(--space-xl);
    border: 1.5px dashed var(--color-border);
    border-radius: var(--radius-lg);
    color: var(--color-muted);
    transition: all var(--transition);
    flex-shrink: 0;
  }
  .drop-zone.compact {
    flex-direction: row;
    padding: var(--space-sm) var(--space-md);
    gap: var(--space-sm);
    font-size: 0.75rem;
    border-radius: var(--radius-md);
  }
  .drop-zone.drag-over {
    border-color: var(--color-accent);
    background: rgba(122, 162, 247, 0.04);
  }
  .drop-title {
    margin: 0;
    font-size: 0.875rem;
    color: var(--color-text);
  }
  .drop-hint {
    margin: 0;
    font-size: 0.75rem;
  }

  /* Sections */
  .section {
    display: flex;
    flex-direction: column;
  }
  .section-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: var(--space-sm);
  }
  h3 {
    margin: 0;
    font-size: 0.8125rem;
    font-weight: 500;
    color: var(--color-muted);
  }
  .section-count {
    font-size: 0.6875rem;
    padding: 1px 7px;
    border-radius: 10px;
    background: var(--color-surface);
    color: var(--color-muted);
    font-family: var(--font-mono);
  }

  .result-summary {
    display: flex;
    gap: var(--space-xs);
  }
  .badge {
    display: inline-flex;
    align-items: center;
    gap: 3px;
    font-size: 0.6875rem;
    padding: 1px 7px;
    border-radius: 10px;
  }
  .badge.success { background: rgba(158, 206, 106, 0.12); color: var(--color-success); }
  .badge.error { background: rgba(247, 118, 142, 0.12); color: var(--color-danger); }

  /* File list */
  .file-list {
    display: flex;
    flex-direction: column;
    gap: var(--space-xs);
  }
  .file-row {
    display: flex;
    align-items: center;
    gap: var(--space-sm);
    padding: var(--space-sm) var(--space-md);
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-sm);
    font-size: 0.8125rem;
    color: var(--color-text);
    transition: border-color var(--transition);
  }
  .file-row:hover { border-color: var(--color-surface-hover); }
  .file-row.has-error { color: var(--color-danger); }
  .file-name {
    flex: 1;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .file-size {
    color: var(--color-muted);
    font-size: 0.6875rem;
    font-family: var(--font-mono);
    flex-shrink: 0;
  }
  .btn-icon {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 20px;
    height: 20px;
    background: none;
    border: none;
    border-radius: var(--radius-sm);
    color: var(--color-muted);
    cursor: pointer;
    transition: all var(--transition);
    flex-shrink: 0;
  }
  .btn-icon:hover { color: var(--color-danger); background: rgba(247, 118, 142, 0.08); }

  .btn-dl {
    display: inline-flex;
    align-items: center;
    gap: 3px;
    padding: 2px 8px;
    background: var(--color-accent);
    color: var(--color-bg);
    border: none;
    border-radius: var(--radius-sm);
    font-size: 0.6875rem;
    font-weight: 500;
    cursor: pointer;
    transition: opacity var(--transition);
    flex-shrink: 0;
  }
  .btn-dl:hover { opacity: 0.85; }

  .error-text {
    font-size: 0.6875rem;
    color: var(--color-danger);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    max-width: 240px;
  }

  /* Convert button */
  .btn-convert {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: var(--space-xs);
    height: 36px;
    padding: 0 var(--space-xl);
    background: var(--color-accent);
    color: var(--color-bg);
    border: none;
    border-radius: var(--radius-md);
    font-size: 0.8125rem;
    font-weight: 500;
    cursor: pointer;
    transition: opacity var(--transition);
    align-self: flex-start;
    flex-shrink: 0;
  }
  .btn-convert:hover:not(:disabled) { opacity: 0.85; }
  .btn-convert:disabled { opacity: 0.5; cursor: not-allowed; }

  :global(.spin) {
    animation: spin 1s linear infinite;
  }
  @keyframes spin {
    from { transform: rotate(0deg); }
    to { transform: rotate(360deg); }
  }
</style>
