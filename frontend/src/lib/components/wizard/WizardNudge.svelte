<script lang="ts">
  import { onMount } from 'svelte';
  import { fade } from 'svelte/transition';
  import type { HintPosition } from './wizard-definitions';

  let {
    anchorSelector,
    position = 'right',
    onactivate,
  }: {
    anchorSelector: string;
    position?: HintPosition;
    onactivate?: () => void;
  } = $props();

  let coords = $state({ top: 0, left: 0 });
  let visible = $state(false);
  let rafId: number;

  function updatePosition() {
    const anchor = document.querySelector(anchorSelector);
    if (!anchor) return;
    const rect = anchor.getBoundingClientRect();
    const offset = 12;

    switch (position) {
      case 'top':
        coords = { top: rect.top - offset, left: rect.left + rect.width / 2 };
        break;
      case 'bottom':
        coords = { top: rect.bottom + offset, left: rect.left + rect.width / 2 };
        break;
      case 'left':
        coords = { top: rect.top + rect.height / 2, left: rect.left - offset };
        break;
      case 'right':
        coords = { top: rect.top + rect.height / 2, left: rect.right + offset };
        break;
    }
  }

  onMount(() => {
    const timer = setTimeout(() => {
      visible = true;
      updatePosition();
    }, 100);

    function tick() {
      if (visible) updatePosition();
      rafId = requestAnimationFrame(tick);
    }
    rafId = requestAnimationFrame(tick);

    return () => {
      clearTimeout(timer);
      cancelAnimationFrame(rafId);
    };
  });
</script>

{#if visible}
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div
    class="wizard-nudge"
    style="top: {coords.top}px; left: {coords.left}px"
    transition:fade={{ duration: 400 }}
    onclick={onactivate}
    onkeydown={(e) => { if (e.key === 'Enter') onactivate?.(); }}
    role="button"
    tabindex="0"
    aria-label="提示可用，点击查看"
  >
    <div class="nudge-dot"></div>
    <div class="nudge-ring"></div>
  </div>
{/if}

<style>
  .wizard-nudge {
    position: fixed;
    z-index: 9998;
    transform: translate(-50%, -50%);
    cursor: pointer;
    pointer-events: auto;
  }

  .nudge-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: var(--color-accent);
    position: relative;
    z-index: 2;
  }

  .nudge-ring {
    position: absolute;
    top: 50%;
    left: 50%;
    width: 22px;
    height: 22px;
    border-radius: 50%;
    border: 1.5px solid var(--color-accent);
    opacity: 0;
    transform: translate(-50%, -50%) scale(0.5);
    animation: wizard-breathe 2.5s ease-in-out infinite;
  }

  @keyframes wizard-breathe {
    0% {
      opacity: 0;
      transform: translate(-50%, -50%) scale(0.5);
    }
    40% {
      opacity: 0.5;
      transform: translate(-50%, -50%) scale(1);
    }
    100% {
      opacity: 0;
      transform: translate(-50%, -50%) scale(1.5);
    }
  }

  @media (prefers-reduced-motion: reduce) {
    .nudge-ring {
      animation: none;
      opacity: 0.35;
      transform: translate(-50%, -50%) scale(1);
    }
  }
</style>
