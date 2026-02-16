<script lang="ts">
  import TemplateSelector from '$lib/components/TemplateSelector.svelte';
  import { convertAndCompile } from '$lib/api/client';
  import { Upload, FileText, Download, X, Loader } from 'lucide-svelte';

  let selectedTemplate = $state('');
  let files: File[] = $state([]);
  let results: { name: string; blob?: Blob; error?: string }[] = $state([]);
  let processing = $state(false);
  let dragOver = $state(false);
  let fileInput: HTMLInputElement;

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

  function download(r: { name: string; blob?: Blob }) {
    if (!r.blob) return;
    const url = URL.createObjectURL(r.blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = r.name;
    a.click();
    URL.revokeObjectURL(url);
  }
</script>

<div class="page">
  <h2>批量转换</h2>

  <div class="controls">
    <TemplateSelector bind:selected={selectedTemplate} />
    <button class="btn-secondary" onclick={() => fileInput?.click()}>
      <Upload size={16} />
      <span>选择文件</span>
    </button>
    <input bind:this={fileInput} type="file" accept=".md,.markdown,.txt" multiple onchange={handleFileInput} hidden />
    <button
      class="btn-primary"
      onclick={convertAll}
      disabled={processing || files.length === 0 || !selectedTemplate}
    >
      {#if processing}
        <Loader size={16} class="spin" />
        <span>转换中...</span>
      {:else}
        <span>转换 {files.length} 个文件</span>
      {/if}
    </button>
  </div>

  <div
    class="drop-zone"
    class:drag-over={dragOver}
    ondrop={handleDrop}
    ondragover={(e) => { e.preventDefault(); dragOver = true; }}
    ondragleave={() => dragOver = false}
    role="region"
    aria-label="拖拽文件区域"
  >
    <Upload size={32} />
    <p>拖拽 Markdown 文件到此处</p>
    <p class="hint">支持 .md .markdown .txt 格式</p>
  </div>

  {#if files.length > 0}
    <div class="section">
      <h3>待转换文件 ({files.length})</h3>
      <ul class="file-list">
        {#each files as file, i (file.name + i)}
          <li>
            <FileText size={16} />
            <span class="file-name">{file.name}</span>
            <span class="file-size">{(file.size / 1024).toFixed(1)} KB</span>
            <button class="btn-icon" onclick={() => removeFile(i)} aria-label="移除 {file.name}">
              <X size={14} />
            </button>
          </li>
        {/each}
      </ul>
    </div>
  {/if}

  {#if results.length > 0}
    <div class="section">
      <h3>转换结果</h3>
      <ul class="file-list">
        {#each results as r (r.name)}
          <li>
            <FileText size={16} />
            <span class="file-name">{r.name}</span>
            {#if r.blob}
              <button class="btn-small" onclick={() => download(r)}>
                <Download size={14} />
                <span>下载</span>
              </button>
            {:else}
              <span class="error">{r.error}</span>
            {/if}
          </li>
        {/each}
      </ul>
    </div>
  {/if}
</div>

<style>
  .page {
    padding: var(--space-xl);
    max-width: 800px;
    margin: 0 auto;
    overflow-y: auto;
    height: 100%;
  }
  h2 {
    margin: 0 0 var(--space-lg);
    font-size: 1.25rem;
  }
  h3 {
    margin: 0 0 var(--space-sm);
    font-size: 0.9375rem;
    color: var(--color-muted);
  }
  .controls {
    display: flex;
    gap: var(--space-sm);
    align-items: center;
    margin-bottom: var(--space-lg);
    flex-wrap: wrap;
  }
  .btn-primary, .btn-secondary {
    display: inline-flex;
    align-items: center;
    gap: var(--space-xs);
    padding: var(--space-sm) var(--space-md);
    border-radius: var(--radius-md);
    font-size: 0.8125rem;
    font-weight: 500;
    cursor: pointer;
    transition: all var(--transition);
    border: none;
  }
  .btn-primary {
    background: var(--color-cta);
    color: white;
  }
  .btn-primary:hover:not(:disabled) { opacity: 0.9; }
  .btn-primary:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
  .btn-secondary {
    background: var(--color-secondary);
    color: var(--color-text);
  }
  .btn-secondary:hover { background: var(--color-surface-hover); }
  .drop-zone {
    border: 2px dashed var(--color-secondary);
    padding: var(--space-2xl);
    text-align: center;
    color: var(--color-muted);
    border-radius: var(--radius-lg);
    transition: all var(--transition);
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--space-sm);
  }
  .drop-zone p { margin: 0; }
  .drop-zone .hint { font-size: 0.75rem; }
  .drag-over {
    border-color: var(--color-cta);
    background: rgba(34, 197, 94, 0.05);
  }
  .section {
    margin-top: var(--space-xl);
  }
  .file-list {
    list-style: none;
    padding: 0;
    margin: 0;
    display: flex;
    flex-direction: column;
    gap: var(--space-xs);
  }
  .file-list li {
    display: flex;
    align-items: center;
    gap: var(--space-sm);
    padding: var(--space-sm) var(--space-md);
    background: var(--color-surface);
    border-radius: var(--radius-md);
    font-size: 0.875rem;
  }
  .file-name { flex: 1; }
  .file-size {
    color: var(--color-muted);
    font-size: 0.75rem;
  }
  .btn-icon {
    background: none;
    border: none;
    color: var(--color-muted);
    padding: var(--space-xs);
    border-radius: var(--radius-sm);
    cursor: pointer;
    transition: color var(--transition);
  }
  .btn-icon:hover { color: var(--color-danger); }
  .btn-small {
    display: inline-flex;
    align-items: center;
    gap: var(--space-xs);
    padding: var(--space-xs) var(--space-sm);
    background: var(--color-cta);
    color: white;
    border: none;
    border-radius: var(--radius-sm);
    font-size: 0.75rem;
    cursor: pointer;
    transition: opacity var(--transition);
  }
  .btn-small:hover { opacity: 0.9; }
  .error {
    color: var(--color-danger);
    font-size: 0.75rem;
  }
</style>
