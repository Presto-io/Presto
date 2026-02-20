<script lang="ts">
  import DOMPurify from 'dompurify';

  let {
    svgPages = [],
    scrollRatio = 0,
    onscroll,
  }: {
    svgPages?: string[];
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
</script>

<div
  bind:this={container}
  class="preview-container"
  role="region"
  aria-label="文档预览"
  onscroll={handleScroll}
>
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
