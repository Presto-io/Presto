<script lang="ts">
  import Editor from '$lib/components/Editor.svelte';
  import Preview from '$lib/components/Preview.svelte';
  import TemplateSelector from '$lib/components/TemplateSelector.svelte';
  import { convert, convertAndCompile } from '$lib/api/client';
  import { Download, Upload } from 'lucide-svelte';

  let markdown = $state('');
  let typstSource = $state('');
  let selectedTemplate = $state('');
  let converting = $state(false);
  let fileInput: HTMLInputElement;
  let debounceTimer: ReturnType<typeof setTimeout>;

  async function handleConvert(md: string) {
    if (!selectedTemplate || !md.trim()) return;
    clearTimeout(debounceTimer);
    debounceTimer = setTimeout(async () => {
      converting = true;
      try {
        typstSource = await convert(md, selectedTemplate);
      } catch (e) {
        console.error('Convert failed:', e);
      } finally {
        converting = false;
      }
    }, 500);
  }

  async function handleDownload() {
    if (!selectedTemplate || !markdown.trim()) return;
    try {
      const blob = await convertAndCompile(markdown, selectedTemplate);
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = 'output.pdf';
      a.click();
      URL.revokeObjectURL(url);
    } catch (e) {
      console.error('Download failed:', e);
    }
  }

  function handleUpload(e: Event) {
    const input = e.target as HTMLInputElement;
    const file = input.files?.[0];
    if (!file) return;
    const reader = new FileReader();
    reader.onload = () => { markdown = reader.result as string; };
    reader.readAsText(file);
  }
</script>

<div class="toolbar">
  <TemplateSelector bind:selected={selectedTemplate} />
  <div class="toolbar-actions">
    <button class="btn-primary" onclick={handleDownload} aria-label="下载 PDF">
      <Download size={16} />
      <span>下载 PDF</span>
    </button>
    <button class="btn-secondary" onclick={() => fileInput?.click()} aria-label="上传 Markdown 文件">
      <Upload size={16} />
      <span>上传 MD</span>
    </button>
    <input bind:this={fileInput} type="file" accept=".md,.markdown,.txt" onchange={handleUpload} hidden />
  </div>
  {#if converting}
    <span class="status">转换中...</span>
  {/if}
</div>

<div class="editor-layout">
  <div class="pane">
    <Editor bind:value={markdown} onchange={handleConvert} />
  </div>
  <div class="divider"></div>
  <div class="pane">
    <Preview {typstSource} />
  </div>
</div>

<style>
  .toolbar {
    display: flex;
    align-items: center;
    gap: var(--space-sm);
    padding: var(--space-sm) var(--space-lg);
    background: var(--color-surface);
    border-bottom: 1px solid var(--color-border);
    flex-shrink: 0;
  }
  .toolbar-actions {
    display: flex;
    gap: var(--space-sm);
    margin-left: auto;
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
  .btn-primary:hover { opacity: 0.9; }
  .btn-secondary {
    background: var(--color-secondary);
    color: var(--color-text);
  }
  .btn-secondary:hover { background: var(--color-surface-hover); }
  .status {
    font-size: 0.75rem;
    color: var(--color-muted);
    margin-left: var(--space-sm);
  }
  .editor-layout {
    display: flex;
    flex: 1;
    overflow: hidden;
  }
  .pane {
    flex: 1;
    overflow: hidden;
  }
  .divider {
    width: 1px;
    background: var(--color-border);
  }
</style>
