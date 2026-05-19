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
    width: 100%;
    height: 100%;
    overflow: hidden;
    background: var(--color-bg);
    cursor: default;
  }
  .showcase-viewport {
    width: 100%;
    min-width: 0;
    max-width: 100%;
    height: 100%;
    overflow: hidden;
    display: flex;
    flex-direction: column;
  }

  .showcase-shell :global(.toolbar) {
    min-width: 0;
    padding-inline: clamp(10px, 2.8vw, var(--space-lg));
  }

  .showcase-shell :global(.toolbar-left),
  .showcase-shell :global(.toolbar-right) {
    min-width: 0;
  }

  .showcase-shell :global(.template-name) {
    max-width: 42vw;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .showcase-shell :global(.editor-layout) {
    width: 100%;
    min-width: 0;
    max-width: 100%;
  }

  .showcase-shell :global(.pane) {
    min-width: 0;
    max-width: 100%;
  }

  .showcase-shell :global(.preview-pane) {
    flex-basis: clamp(230px, 46%, 520px);
  }

  .showcase-shell :global(.divider),
  .showcase-shell :global(.divider-static) {
    width: clamp(3px, 0.9vw, 5px);
  }

  .showcase-shell :global(.editor-container),
  .showcase-shell :global(.editor-container .cm-editor),
  .showcase-shell :global(.editor-container .cm-scroller),
  .showcase-shell :global(.editor-container .cm-content),
  .showcase-shell :global(.preview-container),
  .showcase-shell :global(.svg-pages),
  .showcase-shell :global(.frame-layer),
  .showcase-shell :global(.svg-page) {
    min-width: 0;
    max-width: 100%;
  }

  .showcase-shell :global(.editor-container .cm-line) {
    white-space: pre-wrap;
    overflow-wrap: anywhere;
  }

  .showcase-shell :global(.svg-pages) {
    padding: clamp(10px, 2.4vw, 16px);
    overflow: hidden;
  }

  @media (max-width: 640px) {
    .showcase-shell :global(.toolbar) {
      gap: 8px;
    }

    .showcase-shell :global(.btn-export) {
      padding-inline: 8px;
    }

    .showcase-shell :global(.editor-layout) {
      display: grid;
      grid-template-columns: minmax(0, 1fr) 3px minmax(210px, 0.86fr);
    }

    .showcase-shell :global(.pane),
    .showcase-shell :global(.preview-pane) {
      width: auto;
      min-width: 0;
    }
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
