<script lang="ts">
  import { onMount } from 'svelte';
  import { fly } from 'svelte/transition';
  import { cubicOut } from 'svelte/easing';
  import { X, Lightbulb } from 'lucide-svelte';
  import { dismissPoint } from '$lib/stores/wizard.svelte';
  import type { WizardPointDef } from './wizard-definitions';

  let {
    point,
    ondismiss,
  }: {
    point: WizardPointDef;
    ondismiss?: () => void;
  } = $props();

  let hintEl: HTMLDivElement | undefined = $state();
  let coords = $state({ top: 0, left: 0 });
  let arrowSide = $state<'top' | 'bottom' | 'left' | 'right'>('top');
  let visible = $state(false);
  let rafId: number;

  function computePosition() {
    const anchor = document.querySelector(point.anchorSelector);
    if (!anchor || !hintEl) return;
    const rect = anchor.getBoundingClientRect();
    const hintRect = hintEl.getBoundingClientRect();
    const hintW = hintRect.width || 280;
    const hintH = hintRect.height || 120;
    const gap = 12;
    const vw = window.innerWidth;
    const vh = window.innerHeight;

    let pos = point.position;

    // Fallback if not enough space
    if (pos === 'bottom' && rect.bottom + gap + hintH > vh) pos = 'top';
    if (pos === 'top' && rect.top - gap - hintH < 0) pos = 'bottom';
    if (pos === 'right' && rect.right + gap + hintW > vw) pos = 'left';
    if (pos === 'left' && rect.left - gap - hintW < 0) pos = 'right';

    switch (pos) {
      case 'bottom':
        coords = {
          top: rect.bottom + gap,
          left: Math.max(8, Math.min(vw - hintW - 8, rect.left + rect.width / 2 - hintW / 2)),
        };
        arrowSide = 'top';
        break;
      case 'top':
        coords = {
          top: rect.top - gap - hintH,
          left: Math.max(8, Math.min(vw - hintW - 8, rect.left + rect.width / 2 - hintW / 2)),
        };
        arrowSide = 'bottom';
        break;
      case 'right':
        coords = {
          top: Math.max(8, Math.min(vh - hintH - 8, rect.top + rect.height / 2 - hintH / 2)),
          left: rect.right + gap,
        };
        arrowSide = 'left';
        break;
      case 'left':
        coords = {
          top: Math.max(8, Math.min(vh - hintH - 8, rect.top + rect.height / 2 - hintH / 2)),
          left: rect.left - gap - hintW,
        };
        arrowSide = 'right';
        break;
    }
  }

  function handleDismiss() {
    dismissPoint(point.id);
    ondismiss?.();
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') handleDismiss();
  }

  onMount(() => {
    const timer = setTimeout(() => {
      visible = true;
      requestAnimationFrame(computePosition);
    }, 50);

    function tick() {
      if (visible) computePosition();
      rafId = requestAnimationFrame(tick);
    }
    rafId = requestAnimationFrame(tick);

    document.addEventListener('keydown', handleKeydown);
    return () => {
      clearTimeout(timer);
      cancelAnimationFrame(rafId);
      document.removeEventListener('keydown', handleKeydown);
    };
  });

  const flyParams = $derived({
    y: arrowSide === 'top' ? -8 : arrowSide === 'bottom' ? 8 : 0,
    x: arrowSide === 'left' ? -8 : arrowSide === 'right' ? 8 : 0,
    duration: 300,
    easing: cubicOut,
  });
</script>

{#if visible}
  <div
    bind:this={hintEl}
    class="wizard-hint"
    class:arrow-top={arrowSide === 'top'}
    class:arrow-bottom={arrowSide === 'bottom'}
    class:arrow-left={arrowSide === 'left'}
    class:arrow-right={arrowSide === 'right'}
    style="top: {coords.top}px; left: {coords.left}px"
    transition:fly={flyParams}
    role="tooltip"
    aria-live="polite"
  >
    <div class="hint-header">
      <Lightbulb size={14} />
      <span class="hint-title">{point.title}</span>
      <button class="hint-close" onclick={handleDismiss} aria-label="关闭提示">
        <X size={12} />
      </button>
    </div>
    <p class="hint-body">{point.body}</p>
    {#if point.shortcut}
      <div class="hint-shortcut">
        <kbd>{point.shortcut}</kbd>
      </div>
    {/if}
    <button class="hint-dismiss" onclick={handleDismiss}>知道了</button>
  </div>
{/if}

<style>
  .wizard-hint {
    position: fixed;
    z-index: 9999;
    width: 280px;
    background: var(--color-bg-elevated);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-lg);
    padding: var(--space-md);
    box-shadow: 0 8px 32px rgba(0, 0, 0, 0.5), 0 0 0 1px rgba(122, 162, 247, 0.08);
    font-family: var(--font-ui);
    pointer-events: auto;
  }

  /* Arrow */
  .wizard-hint::before {
    content: '';
    position: absolute;
    width: 8px;
    height: 8px;
    background: var(--color-bg-elevated);
    border: 1px solid var(--color-border);
    transform: rotate(45deg);
  }
  .arrow-top::before {
    top: -5px;
    left: 50%;
    margin-left: -4px;
    border-bottom: none;
    border-right: none;
  }
  .arrow-bottom::before {
    bottom: -5px;
    left: 50%;
    margin-left: -4px;
    border-top: none;
    border-left: none;
  }
  .arrow-left::before {
    left: -5px;
    top: 50%;
    margin-top: -4px;
    border-top: none;
    border-right: none;
  }
  .arrow-right::before {
    right: -5px;
    top: 50%;
    margin-top: -4px;
    border-bottom: none;
    border-left: none;
  }

  .hint-header {
    display: flex;
    align-items: center;
    gap: var(--space-sm);
    margin-bottom: var(--space-sm);
    color: var(--color-accent);
  }

  .hint-title {
    font-size: 0.8125rem;
    font-weight: 600;
    color: var(--color-text-bright);
    flex: 1;
  }

  .hint-close {
    background: none;
    border: none;
    color: var(--color-muted);
    padding: 2px;
    border-radius: var(--radius-sm);
    cursor: pointer;
    transition: color var(--transition);
    display: flex;
    align-items: center;
  }
  .hint-close:hover {
    color: var(--color-text);
  }

  .hint-body {
    margin: 0 0 var(--space-sm);
    font-size: 0.75rem;
    color: var(--color-muted);
    line-height: 1.6;
  }

  .hint-shortcut {
    margin-bottom: var(--space-sm);
  }
  .hint-shortcut kbd {
    display: inline-block;
    padding: 2px 6px;
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: 4px;
    font-family: var(--font-mono);
    font-size: 0.6875rem;
    color: var(--color-text);
    white-space: nowrap;
  }

  .hint-dismiss {
    display: block;
    width: 100%;
    padding: var(--space-xs) 0;
    background: none;
    border: none;
    color: var(--color-accent);
    font-size: 0.75rem;
    font-weight: 500;
    cursor: pointer;
    text-align: center;
    border-top: 1px solid var(--color-border);
    margin-top: var(--space-xs);
    transition: opacity var(--transition);
  }
  .hint-dismiss:hover {
    opacity: 0.8;
  }
</style>
