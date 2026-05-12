<script lang="ts">
  import DOMPurify from 'dompurify';
  import type { PreviewModeState } from '$lib/api/types';

  let {
    svgPages = [],
    modeState,
    debug = false,
    scrollRatio = 0,
    onscroll,
  }: {
    svgPages?: string[];
    modeState?: PreviewModeState;
    debug?: boolean;
    scrollRatio?: number;
    onscroll?: (ratio: number) => void;
  } = $props();

  // SEC-04: Sanitize SVG to prevent XSS via <script>, <foreignObject>, event handlers
  function sanitizeSvg(svg: string): string {
    return DOMPurify.sanitize(svg, {
      USE_PROFILES: { svg: true, svgFilters: true },
      ADD_TAGS: ['use'],
      ADD_ATTR: ['xlink:href', 'clip-path', 'fill-rule', 'transform'],
    });
  }

  let container: HTMLDivElement;
  let ignoreScroll = false;

  // Sync scroll from editor
  $effect(() => {
    if (container && scrollRatio >= 0 && !ignoreScroll) {
      const maxScroll = container.scrollHeight - container.clientHeight;
      if (maxScroll > 0) {
        ignoreScroll = true;
        container.scrollTop = scrollRatio * maxScroll;
        requestAnimationFrame(() => { ignoreScroll = false; });
      }
    }
  });

  function handleScroll() {
    if (ignoreScroll) return;
    const maxScroll = container.scrollHeight - container.clientHeight;
    if (maxScroll > 0 && onscroll) {
      onscroll(container.scrollTop / maxScroll);
    }
  }

  function renderSvgPages(pages: string[]) {
    return pages;
  }
</script>

<div
  bind:this={container}
  class="preview-container"
  role="region"
  aria-label="文档预览"
  onscroll={handleScroll}
>
  {#if !modeState}
    {#if svgPages.length > 0}
      <div class="svg-pages">
        {#each svgPages as svg, i}
          <div class="svg-page" data-page={i + 1}>
            {@html sanitizeSvg(svg)}
          </div>
        {/each}
      </div>
    {:else}
      <div class="placeholder">
        <svg xmlns="http://www.w3.org/2000/svg" width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z"/><polyline points="14 2 14 8 20 8"/></svg>
        <p>在左侧编辑 Markdown，选择模板后预览将在此显示</p>
      </div>
    {/if}
  {:else if modeState.kind === 'embedded'}
    <iframe
      class="embedded-preview"
      src={modeState.dataPlaneUrl}
      sandbox="allow-scripts allow-same-origin"
      title="文档预览"
    ></iframe>
    {#if debug}
      <div class="preview-status">mode embedded · {modeState.sessionId}</div>
    {/if}
  {:else}
    {@const fallbackPages = modeState.kind === 'fallback' ? modeState.svgPages : modeState.fallbackSvgPages}
    {#if modeState.kind === 'starting'}
      <div class="preview-status">兼容预览</div>
    {:else if modeState.kind === 'fallback' && modeState.label}
      <div class="preview-status">{modeState.label}</div>
    {:else if modeState.kind === 'error'}
      <div class="preview-error">{modeState.message}</div>
    {/if}
    {#if fallbackPages.length > 0}
      <div class="svg-pages">
        {#each renderSvgPages(fallbackPages) as svg, i}
          <div class="svg-page" data-page={i + 1}>
            {@html sanitizeSvg(svg)}
          </div>
        {/each}
      </div>
    {:else}
      <div class="placeholder">
        <svg xmlns="http://www.w3.org/2000/svg" width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z"/><polyline points="14 2 14 8 20 8"/></svg>
        <p>在左侧编辑 Markdown，选择模板后预览将在此显示</p>
      </div>
    {/if}
  {/if}
</div>

<style>
  .preview-container {
    height: 100%;
    overflow-y: auto;
    overflow-x: hidden;
    background: var(--color-preview-bg);
    border-left: 1px solid var(--color-border);
  }
  .svg-pages {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 12px;
    padding: 16px;
  }
  .svg-page {
    background: white;
    color: black;
    color-scheme: light;
    box-shadow: var(--shadow-md);
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
  .embedded-preview {
    display: block;
    width: 100%;
    height: 100%;
    border: 0;
    background: white;
  }
  .preview-status,
  .preview-error {
    position: sticky;
    top: 8px;
    z-index: 1;
    width: max-content;
    max-width: calc(100% - 32px);
    margin: 8px 16px 0 auto;
    padding: 4px 8px;
    border-radius: var(--radius-sm);
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    color: var(--color-muted);
    font-size: 12px;
    line-height: 1.4;
  }
  .preview-error {
    color: var(--color-danger);
  }
  .placeholder {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    height: 100%;
    color: var(--color-muted);
    gap: var(--space-md);
    padding: var(--space-xl);
  }
  .placeholder p {
    font-size: 12px;
    margin: 0;
    text-align: center;
    line-height: 1.6;
  }
</style>
