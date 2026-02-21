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

  // Frame thresholds (char index where we switch to next frame)
  const FRAME_THRESHOLDS = [0, 120, 280, 450];

  let typedText = $state('');
  let currentFrame = $state(0);
  let typingTimer: ReturnType<typeof setTimeout>;
  let charIndex = $state(0);

  function sanitizeSvg(svg: string): string {
    return DOMPurify.sanitize(svg, {
      USE_PROFILES: { svg: true, svgFilters: true },
      ADD_TAGS: ['use'],
      ADD_ATTR: ['xlink:href', 'clip-path', 'fill-rule', 'transform'],
    });
  }

  function getTypingDelay(char: string): number {
    // Punctuation pauses
    if (/[。，、；：！？]/.test(char)) return 200 + Math.random() * 200;
    if (char === '\n') return 300 + Math.random() * 200;
    // Normal typing speed
    return 50 + Math.random() * 30;
  }

  function typeNextChar() {
    if (charIndex >= heroTypingContent.length) {
      // Done typing, wait then restart
      typingTimer = setTimeout(() => {
        charIndex = 0;
        typedText = '';
        currentFrame = 0;
        typeNextChar();
      }, 8000);
      return;
    }

    const char = heroTypingContent[charIndex];
    charIndex++;
    typedText = heroTypingContent.slice(0, charIndex);

    // Update preview frame based on character progress
    for (let i = FRAME_THRESHOLDS.length - 1; i >= 0; i--) {
      if (charIndex >= FRAME_THRESHOLDS[i]) {
        currentFrame = i;
        break;
      }
    }

    const delay = getTypingDelay(char);
    typingTimer = setTimeout(typeNextChar, delay);
  }

  onMount(() => {
    // Start typing after brief delay
    typingTimer = setTimeout(typeNextChar, 500);
  });

  onDestroy(() => {
    clearTimeout(typingTimer);
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
</style>
