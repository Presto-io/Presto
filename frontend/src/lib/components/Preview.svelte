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

  const sanitizedSvgCache = new Map<string, string>();

  // SEC-04: Sanitize SVG to prevent XSS via <script>, <foreignObject>, event handlers
  function sanitizeSvg(svg: string): string {
    const cached = sanitizedSvgCache.get(svg);
    if (cached !== undefined) return cached;

    const sanitized = DOMPurify.sanitize(svg, {
      USE_PROFILES: { svg: true, svgFilters: true },
      ADD_TAGS: ['use'],
      ADD_ATTR: ['xlink:href', 'clip-path', 'fill-rule', 'transform'],
    });
    if (sanitizedSvgCache.size > 24) {
      sanitizedSvgCache.clear();
    }
    sanitizedSvgCache.set(svg, sanitized);
    return sanitized;
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

  function embeddedSrc(url: string, reloadKey?: number): string {
    if (!reloadKey) return url;
    const separator = url.includes('?') ? '&' : '?';
    return `${url}${separator}prestoPreviewVersion=${reloadKey}`;
  }

  type FrameSlot = 'a' | 'b';

  let activeFrame = $state<FrameSlot>('a');
  let pendingFrame = $state<FrameSlot | null>(null);
  let frameASrc = $state('');
  let frameBSrc = $state('');
  let frameALoaded = $state(false);
  let frameBLoaded = $state(false);
  let frameALoadToken = 0;
  let frameBLoadToken = 0;

  function frameSrc(frame: FrameSlot): string {
    return frame === 'a' ? frameASrc : frameBSrc;
  }

  function frameLoaded(frame: FrameSlot): boolean {
    return frame === 'a' ? frameALoaded : frameBLoaded;
  }

  function setFrame(frame: FrameSlot, src: string) {
    if (frame === 'a') {
      frameASrc = src;
      frameALoaded = false;
      frameALoadToken += 1;
    } else {
      frameBSrc = src;
      frameBLoaded = false;
      frameBLoadToken += 1;
    }
  }

  function markFrameLoaded(frame: FrameSlot) {
    const token = frame === 'a' ? frameALoadToken : frameBLoadToken;
    requestAnimationFrame(() => {
      requestAnimationFrame(() => {
        const currentToken = frame === 'a' ? frameALoadToken : frameBLoadToken;
        if (token !== currentToken) return;
        if (frame === 'a') {
          frameALoaded = true;
        } else {
          frameBLoaded = true;
        }
        if (pendingFrame === frame) {
          activeFrame = frame;
          pendingFrame = null;
        }
      });
    });
  }

  function activeEmbeddedLoaded(): boolean {
    const src = frameSrc(activeFrame);
    return src !== '' && frameLoaded(activeFrame);
  }

  $effect(() => {
    if (modeState?.kind !== 'embedded') {
      pendingFrame = null;
      frameASrc = '';
      frameBSrc = '';
      frameALoaded = false;
      frameBLoaded = false;
      activeFrame = 'a';
      return;
    }

    const nextSrc = embeddedSrc(modeState.dataPlaneUrl, modeState.reloadKey);
    const currentSrc = frameSrc(activeFrame);
    if (!currentSrc) {
      setFrame(activeFrame, nextSrc);
      pendingFrame = null;
      return;
    }
    if (nextSrc === currentSrc || nextSrc === frameSrc(activeFrame === 'a' ? 'b' : 'a')) {
      return;
    }

    const nextFrame = activeFrame === 'a' ? 'b' : 'a';
    setFrame(nextFrame, nextSrc);
    pendingFrame = nextFrame;
  });
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
    {@const fallbackPages = modeState.fallbackSvgPages}
    {#if !activeEmbeddedLoaded() && fallbackPages.length > 0}
      <div class="embedded-fallback">
        <div class="svg-pages">
          {#each renderSvgPages(fallbackPages) as svg, i}
            <div class="svg-page" data-page={i + 1}>
              {@html sanitizeSvg(svg)}
            </div>
          {/each}
        </div>
      </div>
    {:else if !activeEmbeddedLoaded()}
      <div class="placeholder">
        <svg xmlns="http://www.w3.org/2000/svg" width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z"/><polyline points="14 2 14 8 20 8"/></svg>
        <p>预览加载中</p>
      </div>
    {/if}
    {#if frameASrc}
      <iframe
        class={`embedded-preview ${activeFrame === 'a' && frameALoaded ? 'active' : ''}`}
        src={frameASrc}
        sandbox="allow-scripts allow-same-origin"
        title="文档预览"
        onload={() => markFrameLoaded('a')}
      ></iframe>
    {/if}
    {#if frameBSrc}
      <iframe
        class={`embedded-preview ${activeFrame === 'b' && frameBLoaded ? 'active' : ''}`}
        src={frameBSrc}
        sandbox="allow-scripts allow-same-origin"
        title="文档预览"
        onload={() => markFrameLoaded('b')}
      ></iframe>
    {/if}
    {#if !activeEmbeddedLoaded()}
      <div class="preview-status">预览加载中</div>
    {/if}
    {#if debug}
      <div class="preview-status">mode embedded · {modeState.sessionId}</div>
    {/if}
  {:else}
    {@const fallbackPages = modeState.kind === 'fallback' ? modeState.svgPages : modeState.fallbackSvgPages}
    {#if modeState.kind === 'starting'}
      <div class="preview-status">预览加载中</div>
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
        <p>{modeState.kind === 'starting' ? '预览加载中' : '在左侧编辑 Markdown，选择模板后预览将在此显示'}</p>
      </div>
    {/if}
  {/if}
</div>

<style>
  .preview-container {
    position: relative;
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
    position: absolute;
    inset: 0;
    width: 100%;
    height: 100%;
    border: 0;
    background: white;
    opacity: 0;
    pointer-events: none;
  }
  .embedded-preview.active {
    opacity: 1;
    pointer-events: auto;
  }
  .embedded-fallback {
    position: absolute;
    inset: 0;
    overflow-y: auto;
    background: var(--color-preview-bg);
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
