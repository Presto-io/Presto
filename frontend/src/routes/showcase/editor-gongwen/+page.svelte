<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import Editor from '$lib/components/Editor.svelte';
  import Preview from '$lib/components/Preview.svelte';
  import { Download, Settings, FolderOpen, Layers } from 'lucide-svelte';
  import { gongwenExample } from '$lib/showcase/presets';

  // Load SVG files as raw strings
  import gongwenPage1 from '$lib/showcase/svg/gongwen-page-1.svg?raw';
  import gongwenPage2 from '$lib/showcase/svg/gongwen-page-2.svg?raw';

  let editorValue = $state(gongwenExample);
  let svgPages = $state([gongwenPage1, gongwenPage2]);

  // Scroll sync
  let editorScrollRatio = $state(0);
  let previewScrollRatio = $state(0);
  let scrollSource: 'editor' | 'preview' | null = $state(null);

  // Split pane
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
</script>

<div class="toolbar" onmousemove={handleToolbarMouseMove}>
  <div class="toolbar-left">
    <span class="template-name">类公文模板</span>
    <div class="status-dot"></div>
  </div>
  <div
    class="toolbar-right"
    bind:this={toolbarRightEl}
    onmouseenter={handleToolbarRightEnter}
    onmouseleave={handleToolbarRightLeave}
  >
    <div class="toolbar-hidden-group" class:visible={hiddenButtonsVisible}>
      <button class="btn-toolbar" aria-label="设置">
        <Settings size={14} />
      </button>
      <button class="btn-toolbar" aria-label="打开文件">
        <FolderOpen size={14} />
      </button>
      <button class="btn-toolbar" aria-label="批量转换">
        <Layers size={14} />
      </button>
    </div>
    <button class="btn-export" aria-label="导出 PDF">
      <Download size={14} />
      <span>导出 PDF</span>
    </button>
  </div>
</div>

<div class="editor-layout" bind:this={layoutEl} class:dragging={isDragging}>
  <div class="pane" style="flex: {splitRatio}">
    <Editor bind:value={editorValue} readOnly={true} scrollRatio={editorScrollRatio} onscroll={(ratio: number) => {
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
    <Preview svgPages={svgPages} scrollRatio={previewScrollRatio} onscroll={(ratio: number) => {
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
  .template-name {
    font-size: 13px;
    font-weight: 500;
    color: var(--color-text);
    padding: 4px 10px;
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-sm);
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
  .toolbar-right {
    display: flex;
    align-items: center;
    gap: var(--space-sm);
    margin-left: auto;
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
</style>
