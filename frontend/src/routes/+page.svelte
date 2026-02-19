<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import Editor from '$lib/components/Editor.svelte';
  import Preview from '$lib/components/Preview.svelte';
  import TemplateSelector from '$lib/components/TemplateSelector.svelte';
  import { convert, compile, compileSvg, convertAndCompile, getExample } from '$lib/api/client';
  import { Download, Settings, FolderOpen, Layers } from 'lucide-svelte';
  import { goto } from '$app/navigation';
  import { editor } from '$lib/stores/editor.svelte';
  import { triggerAction, shouldShowPoint } from '$lib/stores/wizard.svelte';

  // Wails runtime bindings (available when running as desktop app)
  declare global {
    interface Window {
      go?: { main: { App: {
        SavePDF: (markdown: string, templateId: string, workDir: string) => Promise<void>;
        OpenFile: () => Promise<{ content: string; dir: string } | null>;
        CompileSVG: (typstSource: string, workDir: string) => Promise<string[]>;
      } } };
      runtime?: { EventsOn: (event: string, cb: (...args: any[]) => void) => void };
    }
  }

  let converting = $state(false);
  let errorMsg = $state('');
  let editorScrollRatio = $state(0);
  let previewScrollRatio = $state(0);
  let scrollSource: 'editor' | 'preview' | null = $state(null);
  let debounceTimer: ReturnType<typeof setTimeout>;

  // Resizable split pane
  let splitRatio = $state(0.5);
  let isDragging = $state(false);
  let layoutEl: HTMLDivElement;

  // Proximity reveal toolbar buttons
  let toolbarRightEl: HTMLDivElement;
  let hiddenButtonsVisible = $state(false);
  let hideTimer: ReturnType<typeof setTimeout>;

  function handleToolbarRightEnter() {
    clearTimeout(hideTimer);
    hiddenButtonsVisible = true;
  }

  function handleToolbarRightLeave() {
    clearTimeout(hideTimer);
    hideTimer = setTimeout(() => { hiddenButtonsVisible = false; }, 800);
  }

  function handleToolbarMouseMove(e: MouseEvent) {
    if (!toolbarRightEl) return;
    const rect = toolbarRightEl.getBoundingClientRect();
    const proximity = 60;
    const inRange = (
      e.clientX >= rect.left - proximity &&
      e.clientX <= rect.right + proximity &&
      e.clientY >= rect.top - proximity &&
      e.clientY <= rect.bottom + proximity
    );
    if (inRange) {
      handleToolbarRightEnter();
    } else if (hiddenButtonsVisible) {
      clearTimeout(hideTimer);
      hideTimer = setTimeout(() => { hiddenButtonsVisible = false; }, 800);
    }
  }

  onDestroy(() => { clearTimeout(hideTimer); });

  function onDividerPointerDown(e: PointerEvent) {
    isDragging = true;
    (e.target as HTMLElement).setPointerCapture(e.pointerId);
  }

  function onDividerPointerMove(e: PointerEvent) {
    if (!isDragging || !layoutEl) return;
    const rect = layoutEl.getBoundingClientRect();
    const ratio = (e.clientX - rect.left) / rect.width;
    splitRatio = Math.min(0.8, Math.max(0.2, ratio));
  }

  function onDividerPointerUp() {
    isDragging = false;
  }

  // Template switching confirmation
  let pendingTemplate = $state('');
  let confirmDialog: HTMLDialogElement;

  async function loadExample(templateId: string) {
    try {
      const example = await getExample(templateId);
      if (example) {
        editor.markdown = example;
        handleConvert(editor.markdown);
      }
    } catch (e) {
      console.error('Failed to load example:', e);
    }
  }

  async function handleTemplateChange(newId: string) {
    if (newId === editor.selectedTemplate) return;

    if (!editor.markdown.trim()) {
      // Empty editor — switch directly and load example
      editor.selectedTemplate = newId;
      await loadExample(newId);
    } else {
      // Has content — show confirmation dialog
      pendingTemplate = newId;
      confirmDialog?.showModal();
    }
  }

  function handleUseExample() {
    confirmDialog?.close();
    editor.selectedTemplate = pendingTemplate;
    loadExample(pendingTemplate);
    pendingTemplate = '';
  }

  function handleKeepContent() {
    confirmDialog?.close();
    editor.selectedTemplate = pendingTemplate;
    handleConvert(editor.markdown);
    pendingTemplate = '';
  }

  function handleCancelSwitch() {
    confirmDialog?.close();
    pendingTemplate = '';
  }

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
          // SEC-26: Escape regex metacharacters to prevent ReDoS
          const escaped = varName.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
          const re = new RegExp(`#let\\s+${escaped}\\s*=\\s*"([^"]*)"`);
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
    if (!editor.selectedTemplate || !md.trim()) return;
    errorMsg = '';

    // Wizard: detect image syntax
    if (md.includes('![') && shouldShowPoint('image-path')) triggerAction('image-path');

    clearTimeout(debounceTimer);
    debounceTimer = setTimeout(async () => {
      converting = true;
      try {
        editor.typstSource = await convert(md, editor.selectedTemplate);
        // Compile to SVG for preview — use Wails binding when available
        // (Wails WebView strips HTTP headers/query params, so workDir gets lost via fetch)
        if (window.go?.main?.App?.CompileSVG) {
          editor.svgPages = await window.go.main.App.CompileSVG(editor.typstSource, editor.documentDir);
        } else {
          editor.svgPages = await compileSvg(editor.typstSource, editor.documentDir || undefined);
        }
      } catch (e) {
        const msg = e instanceof Error ? e.message : String(e);
        console.error('Convert failed:', msg);
        errorMsg = msg;
        // Wizard: detect image-related errors and hint about path rules
        if (/image|图片|not found|file not|读取/.test(msg.toLowerCase())) {
          setTimeout(() => triggerAction('image-error'), 500);
        }
      } finally {
        converting = false;
      }
    }, 500);
  }

  async function handleDownload() {
    if (!editor.selectedTemplate || !editor.markdown.trim()) return;
    errorMsg = '';
    try {
      if (window.go?.main?.App?.SavePDF) {
        await window.go.main.App.SavePDF(editor.markdown, editor.selectedTemplate, editor.documentDir);
        return;
      }
      const blob = await convertAndCompile(editor.markdown, editor.selectedTemplate, editor.documentDir || undefined);
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = extractTypstTitle(editor.typstSource) + '.pdf';
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
          editor.markdown = result.content;
          editor.documentDir = result.dir;
          handleConvert(editor.markdown);
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
        editor.documentDir = '';
        const reader = new FileReader();
        reader.onload = () => {
          editor.markdown = reader.result as string;
          handleConvert(editor.markdown);
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
      window.runtime.EventsOn('menu:templates', () => goto('/settings'));
    }
    // Keyboard shortcut for web: Cmd+, opens settings
    function handleKeydown(e: KeyboardEvent) {
      if ((e.metaKey || e.ctrlKey) && e.key === ',') {
        e.preventDefault();
        goto('/settings');
      }
      if ((e.metaKey || e.ctrlKey) && e.shiftKey && (e.key === 't' || e.key === 'T')) {
        e.preventDefault();
        goto('/settings');
      }
    }
    document.addEventListener('keydown', handleKeydown);
    return () => document.removeEventListener('keydown', handleKeydown);
  });
</script>

<div class="toolbar" style="--wails-draggable:drag" onmousemove={handleToolbarMouseMove}>
  <div class="toolbar-left">
    <TemplateSelector selected={editor.selectedTemplate} onbeforechange={handleTemplateChange} />
    {#if converting}
      <div class="status-dot"></div>
    {/if}
  </div>
  <div
    class="toolbar-right"
    bind:this={toolbarRightEl}
    onmouseenter={handleToolbarRightEnter}
    onmouseleave={handleToolbarRightLeave}
  >
    {#if errorMsg}
      <span class="error-msg" title={errorMsg}>{errorMsg}</span>
    {/if}
    <div class="toolbar-hidden-group" class:visible={hiddenButtonsVisible}>
      <button class="btn-toolbar" onclick={() => goto('/settings')} aria-label="设置" title="设置 (⌘,)">
        <Settings size={14} />
      </button>
      <button class="btn-toolbar" onclick={handleOpen} aria-label="打开文件" title="打开文件 (⌘O)">
        <FolderOpen size={14} />
      </button>
      <button class="btn-toolbar" onclick={() => goto('/batch')} aria-label="批量转换" title="批量转换">
        <Layers size={14} />
      </button>
    </div>
    <button class="btn-export" onclick={handleDownload} aria-label="导出 PDF" title="导出 PDF (⌘E)">
      <Download size={14} />
      <span>导出 PDF</span>
    </button>
  </div>
</div>

<div class="editor-layout" bind:this={layoutEl} class:dragging={isDragging}>
  <div class="pane" style="flex: {splitRatio}">
    <Editor bind:value={editor.markdown} onchange={handleConvert} scrollRatio={editorScrollRatio} onscroll={(ratio: number) => {
      if (scrollSource !== 'preview') {
        scrollSource = 'editor';
        previewScrollRatio = ratio;
        setTimeout(() => { scrollSource = null; }, 100);
      }
    }} />
  </div>
  <div
    class="divider"
    role="separator"
    aria-label="拖动调整宽度"
    aria-orientation="vertical"
    onpointerdown={onDividerPointerDown}
    onpointermove={onDividerPointerMove}
    onpointerup={onDividerPointerUp}
  >
    <div class="divider-grip">
      <span></span><span></span><span></span>
    </div>
  </div>
  <div class="pane" style="flex: {1 - splitRatio}">
    <Preview svgPages={editor.svgPages} scrollRatio={previewScrollRatio} onscroll={(ratio: number) => {
      if (scrollSource !== 'editor') {
        scrollSource = 'preview';
        editorScrollRatio = ratio;
        setTimeout(() => { scrollSource = null; }, 100);
      }
    }} />
  </div>
</div>

<dialog bind:this={confirmDialog} class="confirm-dialog">
  <h3>切换模板</h3>
  <p>当前编辑器中有内容，切换模板后如何处理？</p>
  <div class="dialog-actions">
    <button class="dialog-btn primary" onclick={handleUseExample}>使用示例内容</button>
    <button class="dialog-btn" onclick={handleKeepContent}>保留当前内容</button>
    <button class="dialog-btn" onclick={handleCancelSwitch}>取消</button>
  </div>
</dialog>

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
  .toolbar-hidden-group {
    display: flex;
    align-items: center;
    gap: var(--space-xs);
    opacity: 0;
    pointer-events: none;
    transition: opacity 200ms ease;
  }
  .toolbar-hidden-group.visible {
    opacity: 1;
    pointer-events: auto;
  }
  .btn-toolbar {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 28px;
    height: 28px;
    background: var(--color-surface);
    color: var(--color-text);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-sm);
    cursor: pointer;
    transition: all var(--transition);
    flex-shrink: 0;
  }
  .btn-toolbar:hover {
    background: var(--color-surface-hover);
    border-color: var(--color-muted);
    color: var(--color-accent);
  }
  @media (hover: none) {
    .toolbar-hidden-group {
      opacity: 1;
      pointer-events: auto;
    }
  }
  .btn-export {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    height: 28px;
    padding: 0 10px;
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
  .editor-layout.dragging {
    cursor: col-resize;
    user-select: none;
  }
  .pane {
    overflow: hidden;
    min-width: 0;
  }
  .divider {
    width: 5px;
    flex-shrink: 0;
    background: var(--color-border);
    cursor: col-resize;
    transition: background 0.15s, width 0.15s;
    touch-action: none;
    position: relative;
    display: flex;
    align-items: center;
    justify-content: center;
  }
  .divider:hover,
  .editor-layout.dragging .divider {
    background: var(--color-accent);
    width: 7px;
  }
  .divider-grip {
    display: flex;
    flex-direction: column;
    gap: 3px;
    opacity: 0;
    transition: opacity 0.2s;
    pointer-events: none;
  }
  .divider:hover .divider-grip,
  .editor-layout.dragging .divider-grip {
    opacity: 0.9;
  }
  .divider-grip span {
    width: 3px;
    height: 3px;
    border-radius: 50%;
    background: var(--color-bg);
  }
  .confirm-dialog {
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md, 8px);
    background: var(--color-surface);
    color: var(--color-text);
    padding: 24px;
    max-width: 400px;
    font-family: var(--font-ui);
  }
  .confirm-dialog::backdrop {
    background: rgba(0, 0, 0, 0.4);
  }
  .confirm-dialog h3 {
    margin: 0 0 8px;
    font-size: 16px;
    font-weight: 600;
  }
  .confirm-dialog p {
    margin: 0 0 20px;
    font-size: 13px;
    color: var(--color-muted);
    line-height: 1.5;
  }
  .dialog-actions {
    display: flex;
    gap: 8px;
    justify-content: flex-end;
  }
  .dialog-btn {
    padding: 6px 14px;
    border: 1px solid var(--color-border);
    border-radius: var(--radius-sm);
    background: var(--color-surface);
    color: var(--color-text);
    font-size: 12px;
    cursor: pointer;
    transition: opacity var(--transition);
  }
  .dialog-btn:hover { opacity: 0.85; }
  .dialog-btn.primary {
    background: var(--color-accent);
    color: var(--color-bg);
    border-color: var(--color-accent);
  }
</style>
