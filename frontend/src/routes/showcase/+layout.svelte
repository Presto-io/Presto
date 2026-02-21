<script lang="ts">
  import '../../app.css';
  import { onMount } from 'svelte';
  import { page } from '$app/stores';

  let { children } = $props();

  // Whitelist selectors for allowed interactions
  const INTERACTIVE_SELECTORS = [
    '.divider', '.divider-grip',
    '.keyword-chip',
    '.batch-file-row', '.drag-handle',
    '.cm-content', '.cm-scroller', '.cm-line', '.cm-cursor', '.cm-selectionBackground',
    '.preview-container', '.preview-scroll', '.svg-pages', '.svg-page',
  ];

  function isInteractive(target: EventTarget | null): boolean {
    if (!(target instanceof Element)) return false;
    return INTERACTIVE_SELECTORS.some(sel => target.closest(sel) !== null);
  }

  onMount(() => {
    function interceptClick(e: Event) {
      if (isInteractive(e.target)) return;
      e.preventDefault();
      e.stopPropagation();
    }

    function interceptKeydown(e: Event) {
      // Allow tab navigation but block all other keyboard input
      e.preventDefault();
      e.stopPropagation();
    }

    function interceptContextMenu(e: Event) {
      e.preventDefault();
      e.stopPropagation();
    }

    // Capture phase interception
    window.addEventListener('click', interceptClick, true);
    window.addEventListener('mousedown', interceptClick, true);
    window.addEventListener('keydown', interceptKeydown, true);
    window.addEventListener('contextmenu', interceptContextMenu, true);

    return () => {
      window.removeEventListener('click', interceptClick, true);
      window.removeEventListener('mousedown', interceptClick, true);
      window.removeEventListener('keydown', interceptKeydown, true);
      window.removeEventListener('contextmenu', interceptContextMenu, true);
    };
  });
</script>

<div class="showcase-shell">
  <div class="showcase-viewport">
    {@render children()}
  </div>
</div>

<style>
  .showcase-shell {
    width: 100vw;
    height: 100vh;
    overflow: hidden;
    background: var(--color-bg);
    cursor: default;
  }
  .showcase-viewport {
    width: 1200px;
    height: 800px;
    transform-origin: top left;
    overflow: hidden;
    display: flex;
    flex-direction: column;
  }

  /* Override cursor for interactive areas */
  .showcase-shell :global(.divider) {
    cursor: col-resize !important;
  }
  .showcase-shell :global(.keyword-chip) {
    cursor: pointer !important;
  }
  .showcase-shell :global(.drag-handle) {
    cursor: grab !important;
  }
  .showcase-shell :global(.cm-content) {
    cursor: text !important;
  }
  .showcase-shell :global(.preview-container),
  .showcase-shell :global(.cm-scroller) {
    cursor: default !important;
  }
</style>
