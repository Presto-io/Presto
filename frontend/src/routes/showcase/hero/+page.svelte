<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import Editor from '$lib/components/Editor.svelte';
  import DOMPurify from 'dompurify';
  import { heroTypingContent } from '$lib/showcase/presets';
  import { Download } from 'lucide-svelte';

  import heroFrame0 from '$lib/showcase/hero-frames/hero-frame-0.svg?raw';
  import heroFrame1 from '$lib/showcase/hero-frames/hero-frame-1.svg?raw';
  import heroFrame2 from '$lib/showcase/hero-frames/hero-frame-2.svg?raw';
  import heroFrame3 from '$lib/showcase/hero-frames/hero-frame-3.svg?raw';

  const frames = [heroFrame0, heroFrame1, heroFrame2, heroFrame3];

  // Sections: each section appears as a whole block, then triggers a frame switch
  // Order: empty → frontmatter → frame-0 → title → frame-1 → paragraph → frame-2 → rest → frame-3
  const SECTIONS = [
    { end: 114, frame: 0, delay: 800 },   // frontmatter → frame-0
    { end: 124, frame: 1, delay: 600 },   // 各部门、各单位： → frame-1
    { end: 206, frame: 2, delay: 600 },   // 第一段正文 → frame-2
    { end: heroTypingContent.length, frame: 3, delay: 0 },  // 剩余全部 → frame-3
  ];

  let typedText = $state('');
  let currentFrame = $state(-1);  // -1 = no SVG shown
  let animTimer: ReturnType<typeof setTimeout>;
  let sectionIndex = 0;

  function sanitizeSvg(svg: string): string {
    return DOMPurify.sanitize(svg, {
      USE_PROFILES: { svg: true, svgFilters: true },
      ADD_TAGS: ['use'],
      ADD_ATTR: ['xlink:href', 'clip-path', 'fill-rule', 'transform'],
    });
  }

  function showNextSection() {
    if (sectionIndex >= SECTIONS.length) {
      // All done, wait then restart
      animTimer = setTimeout(() => {
        sectionIndex = 0;
        typedText = '';
        currentFrame = -1;
        animTimer = setTimeout(showNextSection, 500);
      }, 8000);
      return;
    }

    const section = SECTIONS[sectionIndex];
    typedText = heroTypingContent.slice(0, section.end);
    currentFrame = section.frame;
    sectionIndex++;

    animTimer = setTimeout(showNextSection, section.delay);
  }

  onMount(() => {
    animTimer = setTimeout(showNextSection, 500);
  });

  onDestroy(() => {
    clearTimeout(animTimer);
  });
</script>

<div class="toolbar">
  <div class="toolbar-left">
    <span class="template-name">类公文模板</span>
    <div class="status-dot"></div>
  </div>
  <div class="toolbar-right">
    <button class="btn-export" aria-label="导出 PDF">
      <Download size={14} />
      <span>导出 PDF</span>
    </button>
  </div>
</div>

<div class="editor-layout">
  <div class="pane">
    <Editor value={typedText} readOnly={true} />
  </div>
  <div class="divider-static">
    <div class="divider-grip">
      <span></span><span></span><span></span>
    </div>
  </div>
  <div class="pane preview-pane">
    <div class="preview-container">
      {#if currentFrame >= 0}
        <div class="svg-pages">
          {#each frames as frame, i}
            <div
              class="frame-layer"
              class:active={i === currentFrame}
            >
              <div class="svg-page">
                {@html sanitizeSvg(frame)}
              </div>
            </div>
          {/each}
        </div>
      {:else}
        <div class="preview-empty">
          <svg xmlns="http://www.w3.org/2000/svg" width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z"/><polyline points="14 2 14 8 20 8"/></svg>
          <p>在左侧编辑 Markdown，选择模板后预览将在此显示</p>
        </div>
      {/if}
    </div>
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
  .pane {
    flex: 1;
    overflow: hidden;
    min-width: 0;
  }
  .divider-static {
    width: 5px;
    flex-shrink: 0;
    background: var(--color-border);
    display: flex;
    align-items: center;
    justify-content: center;
    position: relative;
  }
  .divider-grip {
    display: flex;
    flex-direction: column;
    gap: 3px;
    opacity: 0.3;
    pointer-events: none;
  }
  .divider-grip span {
    width: 3px;
    height: 3px;
    border-radius: 50%;
    background: var(--color-bg);
  }
  .preview-pane {
    position: relative;
  }
  .preview-container {
    height: 100%;
    overflow-y: auto;
    overflow-x: hidden;
    background: var(--color-preview-bg);
    border-left: 1px solid var(--color-border);
  }
  .svg-pages {
    position: relative;
    padding: 16px;
    display: flex;
    flex-direction: column;
    align-items: center;
  }
  .frame-layer {
    position: absolute;
    inset: 16px;
    display: flex;
    justify-content: center;
    opacity: 0;
    transition: opacity 300ms ease;
    pointer-events: none;
  }
  .frame-layer.active {
    opacity: 1;
    pointer-events: auto;
    position: relative;
    inset: 0;
  }
  .svg-page {
    background: white;
    color: black;
    color-scheme: light;
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.4);
    border-radius: 2px;
    overflow: hidden;
    width: 100%;
    max-width: 100%;
  }
  .svg-page :global(svg) {
    display: block;
    width: 100%;
    height: auto;
  }
  .preview-empty {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    height: 100%;
    color: var(--color-muted);
    gap: var(--space-md);
    padding: var(--space-xl);
  }
  .preview-empty p {
    font-size: 12px;
    margin: 0;
    text-align: center;
    line-height: 1.6;
  }
</style>
