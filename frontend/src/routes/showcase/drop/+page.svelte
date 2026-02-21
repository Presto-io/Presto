<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { FileText } from 'lucide-svelte';
  import Editor from '$lib/components/Editor.svelte';
  import Preview from '$lib/components/Preview.svelte';
  import { gongwenExample } from '$lib/showcase/presets';
  import gongwenPage1 from '$lib/showcase/svg/gongwen-page-1.svg?raw';

  let editorValue = $state(gongwenExample);
  let svgPages = $state([gongwenPage1]);

  // Animation state
  let phase = $state<'editor' | 'flying' | 'overlay' | 'idle'>('idle');
  let animTimer: ReturnType<typeof setTimeout>;

  function startCycle() {
    // Phase 1: show editor (already visible)
    phase = 'editor';

    animTimer = setTimeout(() => {
      // Phase 2: file icon flies in
      phase = 'flying';

      animTimer = setTimeout(() => {
        // Phase 3: drop overlay appears
        phase = 'overlay';

        animTimer = setTimeout(() => {
          // Phase 4: overlay fades out, idle
          phase = 'idle';

          animTimer = setTimeout(() => {
            // Restart cycle
            startCycle();
          }, 5000);
        }, 3000);
      }, 800); // flight duration
    }, 1000);
  }

  onMount(() => {
    startCycle();
  });

  onDestroy(() => {
    clearTimeout(animTimer);
  });
</script>

<div class="drop-showcase">
  <!-- Background: editor interface (simplified) -->
  <div class="editor-bg">
    <div class="toolbar">
      <span class="template-name">类公文模板</span>
    </div>
    <div class="editor-layout">
      <div class="pane">
        <Editor value={editorValue} readOnly={true} />
      </div>
      <div class="divider-static"></div>
      <div class="pane">
        <Preview svgPages={svgPages} />
      </div>
    </div>
  </div>

  <!-- Flying file icon -->
  {#if phase === 'flying' || phase === 'overlay'}
    <div class="file-icon" class:landed={phase === 'overlay'}>
      <FileText size={48} strokeWidth={1.5} />
      <span class="file-label">报告.md</span>
    </div>
  {/if}

  <!-- Drop overlay -->
  {#if phase === 'overlay'}
    <div class="drop-overlay" class:fade-in={true}>
      <div class="drop-content">
        <FileText size={32} />
        <span>释放以导入文件</span>
      </div>
    </div>
  {/if}
</div>

<style>
  .drop-showcase {
    width: 100%;
    height: 100%;
    position: relative;
    overflow: hidden;
  }

  .editor-bg {
    width: 100%;
    height: 100%;
    display: flex;
    flex-direction: column;
    opacity: 0.85;
  }

  .toolbar {
    display: flex;
    align-items: center;
    padding: var(--space-sm) var(--space-lg);
    background: var(--color-bg);
    border-bottom: 1px solid var(--color-border);
    flex-shrink: 0;
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
  }

  /* Flying file icon */
  .file-icon {
    position: absolute;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 4px;
    color: var(--color-accent);
    z-index: 100;
    animation: fly-in 800ms cubic-bezier(0.16, 1, 0.3, 1) forwards;
    filter: drop-shadow(0 4px 12px rgba(0, 0, 0, 0.3));
  }

  .file-icon.landed {
    animation: none;
    top: 50%;
    left: 50%;
    transform: translate(-50%, -50%);
    opacity: 0;
  }

  .file-label {
    font-size: 11px;
    font-weight: 500;
    background: var(--color-surface);
    padding: 2px 8px;
    border-radius: var(--radius-sm);
    border: 1px solid var(--color-border);
    white-space: nowrap;
  }

  @keyframes fly-in {
    0% {
      top: -10%;
      right: 5%;
      left: auto;
      opacity: 0;
      transform: scale(0.6) rotate(-15deg);
    }
    30% {
      opacity: 1;
    }
    100% {
      top: 50%;
      left: 50%;
      right: auto;
      transform: translate(-50%, -50%) scale(1) rotate(0deg);
      opacity: 1;
    }
  }

  /* Drop overlay */
  .drop-overlay {
    position: absolute;
    inset: 0;
    z-index: 90;
    background: var(--color-overlay-bg);
    display: flex;
    align-items: center;
    justify-content: center;
    animation: overlay-fade-in 300ms ease-out;
  }

  .drop-content {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--space-md);
    padding: var(--space-2xl);
    border: 2px dashed var(--color-accent);
    border-radius: var(--radius-lg);
    color: var(--color-accent);
    font-size: 1rem;
    font-weight: 500;
  }

  @keyframes overlay-fade-in {
    from { opacity: 0; }
    to { opacity: 1; }
  }
</style>
