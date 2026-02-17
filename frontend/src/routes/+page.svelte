<script lang="ts">
  import Editor from '$lib/components/Editor.svelte';
  import Preview from '$lib/components/Preview.svelte';
  import TemplateSelector from '$lib/components/TemplateSelector.svelte';
  import { convert, compile, convertAndCompile } from '$lib/api/client';
  import { Download, Upload } from 'lucide-svelte';

  // Wails runtime bindings (available when running as desktop app)
  declare global {
    interface Window {
      go?: { main: { App: { SavePDF: (markdown: string, templateId: string) => Promise<void> } } };
    }
  }

  let markdown = $state('');
  let typstSource = $state('');
  let previewUrl = $state('');
  let selectedTemplate = $state('');
  let converting = $state(false);
  let errorMsg = $state('');
  let fileInput: HTMLInputElement;
  let debounceTimer: ReturnType<typeof setTimeout>;

  function extractTypstTitle(typ: string): string {
    const lines = typ.split('\n');
    for (let level = 1; level <= 5; level++) {
      const prefix = '='.repeat(level) + ' ';
      const deeper = '='.repeat(level + 1);
      for (const line of lines) {
        const trimmed = line.trim();
        if (!trimmed.startsWith(prefix)) continue;
        if (level < 5 && trimmed.startsWith(deeper)) continue;
        let content = trimmed.slice(prefix.length).trim();
        // Resolve variable references like #autoTitle.split(...)
        if (content.startsWith('#')) {
          let varName = content.slice(1);
          const cut = varName.search(/[.( ]/);
          if (cut > 0) varName = varName.slice(0, cut);
          const re = new RegExp(`#let\\s+${varName}\\s*=\\s*"([^"]*)"`);
          for (const l of lines) {
            const m = l.match(re);
            if (m) { content = m[1]; break; }
          }
          if (content.startsWith('#')) continue; // couldn't resolve
        }
        const title = content.trim().replace(/[/\\:*?"<>|]/g, '_');
        if (title) return title;
      }
    }
    return 'output';
  }

  async function handleConvert(md: string) {
    if (!selectedTemplate || !md.trim()) return;
    errorMsg = '';
    clearTimeout(debounceTimer);
    debounceTimer = setTimeout(async () => {
      converting = true;
      try {
        typstSource = await convert(md, selectedTemplate);
        // Compile typst to PDF for preview
        const blob = await compile(typstSource);
        if (previewUrl) URL.revokeObjectURL(previewUrl);
        previewUrl = URL.createObjectURL(blob);
      } catch (e) {
        const msg = e instanceof Error ? e.message : String(e);
        console.error('Convert failed:', msg);
        errorMsg = `转换失败: ${msg}`;
      } finally {
        converting = false;
      }
    }, 500);
  }

  async function handleDownload() {
    if (!selectedTemplate || !markdown.trim()) return;
    errorMsg = '';
    try {
      // Use Wails native save dialog when available (desktop app)
      if (window.go?.main?.App?.SavePDF) {
        await window.go.main.App.SavePDF(markdown, selectedTemplate);
        return;
      }
      // Fallback for browser mode
      const blob = await convertAndCompile(markdown, selectedTemplate);
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = extractTypstTitle(typstSource) + '.pdf';
      a.click();
      URL.revokeObjectURL(url);
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e);
      console.error('Download failed:', msg);
      errorMsg = `导出失败: ${msg}`;
    }
  }

  function handleUpload(e: Event) {
    const input = e.target as HTMLInputElement;
    const file = input.files?.[0];
    if (!file) return;
    const reader = new FileReader();
    reader.onload = () => {
      markdown = reader.result as string;
      handleConvert(markdown);
    };
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
  {#if errorMsg}
    <span class="error-msg">{errorMsg}</span>
  {/if}
</div>

<div class="editor-layout">
  <div class="pane">
    <Editor bind:value={markdown} onchange={handleConvert} />
  </div>
  <div class="divider"></div>
  <div class="pane">
    <Preview {previewUrl} />
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
  .error-msg {
    font-size: 0.75rem;
    color: #ef4444;
    margin-left: var(--space-sm);
    max-width: 400px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
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
