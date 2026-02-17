<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import Editor from '$lib/components/Editor.svelte';
  import Preview from '$lib/components/Preview.svelte';
  import TemplateSelector from '$lib/components/TemplateSelector.svelte';
  import { convert, compile, compileSvg, convertAndCompile } from '$lib/api/client';
  import { Download } from 'lucide-svelte';
  import { goto } from '$app/navigation';

  // Wails runtime bindings (available when running as desktop app)
  declare global {
    interface Window {
      go?: { main: { App: {
        SavePDF: (markdown: string, templateId: string, workDir: string) => Promise<void>;
        OpenFile: () => Promise<{ content: string; dir: string } | null>;
      } } };
      runtime?: { EventsOn: (event: string, cb: (...args: any[]) => void) => void };
    }
  }

  let markdown = $state('');
  let typstSource = $state('');
  let svgPages: string[] = $state([]);
  let selectedTemplate = $state('');
  let converting = $state(false);
  let errorMsg = $state('');
  let editorScrollRatio = $state(0);
  let previewScrollRatio = $state(0);
  let scrollSource: 'editor' | 'preview' | null = $state(null);
  let documentDir = $state('');
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
        if (content.startsWith('#')) {
          let varName = content.slice(1);
          const cut = varName.search(/[.( ]/);
          if (cut > 0) varName = varName.slice(0, cut);
          const re = new RegExp(`#let\\s+${varName}\\s*=\\s*"([^"]*)"`);
          for (const l of lines) {
            const m = l.match(re);
            if (m) { content = m[1]; break; }
          }
          if (content.startsWith('#')) continue;
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
        // Compile to SVG for preview
        svgPages = await compileSvg(typstSource, documentDir || undefined);
      } catch (e) {
        const msg = e instanceof Error ? e.message : String(e);
        console.error('Convert failed:', msg);
        errorMsg = msg;
      } finally {
        converting = false;
      }
    }, 500);
  }

  async function handleDownload() {
    if (!selectedTemplate || !markdown.trim()) return;
    errorMsg = '';
    try {
      if (window.go?.main?.App?.SavePDF) {
        await window.go.main.App.SavePDF(markdown, selectedTemplate, documentDir);
        return;
      }
      const blob = await convertAndCompile(markdown, selectedTemplate, documentDir || undefined);
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = extractTypstTitle(typstSource) + '.pdf';
      a.click();
      URL.revokeObjectURL(url);
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e);
      console.error('Download failed:', msg);
      errorMsg = msg;
    }
  }

  async function handleOpen() {
    try {
      if (window.go?.main?.App?.OpenFile) {
        const result = await window.go.main.App.OpenFile();
        if (result) {
          markdown = result.content;
          documentDir = result.dir;
          handleConvert(markdown);
        }
        return;
      }
      // Browser fallback: use file input
      const input = document.createElement('input');
      input.type = 'file';
      input.accept = '.md,.markdown,.txt';
      input.onchange = () => {
        const file = input.files?.[0];
        if (!file) return;
        // Browser File API doesn't expose directory path
        documentDir = '';
        const reader = new FileReader();
        reader.onload = () => {
          markdown = reader.result as string;
          handleConvert(markdown);
        };
        reader.readAsText(file);
      };
      input.click();
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e);
      errorMsg = msg;
    }
  }

  onMount(() => {
    // Listen for Wails menu events
    if (window.runtime?.EventsOn) {
      window.runtime.EventsOn('menu:open', handleOpen);
      window.runtime.EventsOn('menu:export', handleDownload);
      window.runtime.EventsOn('menu:settings', () => goto('/settings'));
    }
    // Keyboard shortcut for web: Cmd+, opens settings
    function handleKeydown(e: KeyboardEvent) {
      if ((e.metaKey || e.ctrlKey) && e.key === ',') {
        e.preventDefault();
        goto('/settings');
      }
    }
    document.addEventListener('keydown', handleKeydown);
    return () => document.removeEventListener('keydown', handleKeydown);
  });
</script>

<div class="toolbar" style="--wails-draggable:drag">
  <div class="toolbar-left">
    <TemplateSelector bind:selected={selectedTemplate} />
    {#if converting}
      <div class="status-dot"></div>
    {/if}
  </div>
  <div class="toolbar-right">
    {#if errorMsg}
      <span class="error-msg" title={errorMsg}>{errorMsg}</span>
    {/if}
    <button class="btn-export" onclick={handleDownload} aria-label="导出 PDF" title="导出 PDF (⌘E)">
      <Download size={14} />
      <span>导出</span>
    </button>
  </div>
</div>

<div class="editor-layout">
  <div class="pane">
    <Editor bind:value={markdown} onchange={handleConvert} scrollRatio={editorScrollRatio} onscroll={(ratio: number) => {
      if (scrollSource !== 'preview') {
        scrollSource = 'editor';
        previewScrollRatio = ratio;
        setTimeout(() => { scrollSource = null; }, 100);
      }
    }} />
  </div>
  <div class="pane">
    <Preview {svgPages} scrollRatio={previewScrollRatio} onscroll={(ratio: number) => {
      if (scrollSource !== 'editor') {
        scrollSource = 'preview';
        editorScrollRatio = ratio;
        setTimeout(() => { scrollSource = null; }, 100);
      }
    }} />
  </div>
</div>

<style>
  .toolbar {
    display: flex;
    align-items: center;
    gap: var(--space-sm);
    padding: var(--space-sm) var(--space-lg);
    padding-top: 38px;
    background: var(--color-bg);
    border-bottom: 1px solid var(--color-border);
    flex-shrink: 0;
    min-height: 0;
  }
  .toolbar-left {
    display: flex;
    align-items: center;
    gap: var(--space-md);
  }
  .status-dot {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: var(--color-accent);
    animation: pulse 1s ease-in-out infinite;
  }
  @keyframes pulse {
    0%, 100% { opacity: 0.4; }
    50% { opacity: 1; }
  }
  .error-msg {
    font-size: 12px;
    color: var(--color-danger);
    max-width: 300px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .toolbar-right {
    display: flex;
    align-items: center;
    gap: var(--space-sm);
    margin-left: auto;
    -webkit-app-region: no-drag;
  }
  .btn-export {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    padding: 4px 10px;
    background: var(--color-accent);
    color: var(--color-bg);
    border: none;
    border-radius: var(--radius-sm);
    font-size: 12px;
    font-weight: 500;
    cursor: pointer;
    transition: opacity var(--transition);
  }
  .btn-export:hover { opacity: 0.85; }
  .editor-layout {
    display: flex;
    flex: 1;
    overflow: hidden;
  }
  .pane {
    flex: 1;
    overflow: hidden;
  }
</style>
