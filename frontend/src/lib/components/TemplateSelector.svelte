<script lang="ts">
  import { onMount } from 'svelte';
  import { fly } from 'svelte/transition';
  import { cubicOut } from 'svelte/easing';
  import { ChevronDown, Search, AlertTriangle, ShoppingBag } from 'lucide-svelte';
  import { goto } from '$app/navigation';
  import { templateStore } from '$lib/stores/templates.svelte';
  import { firstLaunchStore } from '$lib/stores/first-launch.svelte';
  import { editor } from '$lib/stores/editor.svelte';

  let {
    selected = $bindable(''),
    onbeforechange
  }: {
    selected?: string;
    onbeforechange?: (newValue: string) => void;
  } = $props();

  let templates = $derived(templateStore.templates);
  let loading = $derived(templateStore.loading || firstLaunchStore.isActive);
  let disabled = $derived(loading);
  let open = $state(false);
  let search = $state('');
  let highlightIndex = $state(0);
  let wrapperEl: HTMLDivElement;
  let triggerEl: HTMLButtonElement;
  let searchInputEl: HTMLInputElement | undefined = $state();

  let showSearch = $derived(templates.length >= 7);

  let filtered = $derived(
    templates.filter(tpl => {
      if (!search) return true;
      const q = search.toLowerCase();
      return (tpl.displayName || tpl.name).toLowerCase().includes(q) ||
        tpl.name.toLowerCase().includes(q);
    })
  );

  let selectedDisplay = $derived(
    loading ? '加载中...' :
    templates.find(t => t.name === selected)?.displayName ||
    templates.find(t => t.name === selected)?.name ||
    selected ||
    '选择模板'
  );

  let selectedHasMissing = $derived(
    (templates.find(t => t.name === selected)?.missingFonts?.length ?? 0) > 0
  );

  onMount(() => {
    templateStore.load().then(() => {
      if (!selected && templates.length > 0 && !editor.markdown.trim() && !editor.pendingExternalLoad) {
        if (onbeforechange) {
          onbeforechange(templates[0].name);
        } else {
          selected = templates[0].name;
        }
      }
    });

    document.addEventListener('pointerdown', handleOutsideClick, true);
    return () => document.removeEventListener('pointerdown', handleOutsideClick, true);
  });

  function toggle() {
    if (disabled) return;
    open = !open;
    if (open) {
      search = '';
      highlightIndex = Math.max(0, filtered.findIndex(t => t.name === selected));
      requestAnimationFrame(() => searchInputEl?.focus());
    }
  }

  function selectTemplate(name: string) {
    open = false;
    search = '';
    if (onbeforechange) {
      onbeforechange(name);
    } else {
      selected = name;
    }
    triggerEl?.focus();
  }

  function handleKeydown(e: KeyboardEvent) {
    if (!open) {
      if (e.key === 'Enter' || e.key === ' ' || e.key === 'ArrowDown') {
        e.preventDefault();
        toggle();
      }
      return;
    }
    switch (e.key) {
      case 'ArrowDown':
        e.preventDefault();
        highlightIndex = Math.min(highlightIndex + 1, filtered.length - 1);
        break;
      case 'ArrowUp':
        e.preventDefault();
        highlightIndex = Math.max(highlightIndex - 1, 0);
        break;
      case 'Enter':
        e.preventDefault();
        if (filtered[highlightIndex]) {
          selectTemplate(filtered[highlightIndex].name);
        }
        break;
      case 'Escape':
        e.preventDefault();
        open = false;
        triggerEl?.focus();
        break;
    }
  }

  function handleOutsideClick(e: PointerEvent) {
    if (open && wrapperEl && !wrapperEl.contains(e.target as Node)) {
      open = false;
    }
  }

  $effect(() => {
    // Reset highlight when search changes
    void search;
    highlightIndex = 0;
  });
</script>

<div class="selector-wrapper" bind:this={wrapperEl} onkeydown={handleKeydown} role="combobox" aria-expanded={open} aria-controls="selector-listbox" tabindex="0">
  <button
    bind:this={triggerEl}
    class="selector-trigger"
    class:loading
    class:open
    onclick={toggle}
    aria-haspopup="listbox"
    aria-expanded={open}
    aria-label="选择模板"
    {disabled}
  >
    <span class="selector-label">{selectedDisplay}</span>
    {#if loading}
      <span class="loading-dot"></span>
    {:else if selectedHasMissing}
      <span class="selector-warn" title="缺少字体"><AlertTriangle size={11} /></span>
    {/if}
    {#if !loading}
      <ChevronDown size={12} />
    {/if}
  </button>

  {#if open}
    <div
      class="selector-dropdown" id="selector-listbox"
      role="listbox"
      transition:fly={{ y: -4, duration: 150, easing: cubicOut }}
    >
      {#if showSearch}
        <div class="selector-search">
          <Search size={12} />
          <input
            bind:this={searchInputEl}
            type="text"
            placeholder="搜索模板…"
            bind:value={search}
          />
        </div>
      {/if}
      <div class="selector-options">
        {#each filtered as tpl, i (tpl.name)}
          <button
            class="selector-option"
            class:selected={tpl.name === selected}
            class:highlighted={i === highlightIndex}
            role="option"
            aria-selected={tpl.name === selected}
            onclick={() => selectTemplate(tpl.name)}
            onpointerenter={() => highlightIndex = i}
          >
            <span class="option-label">{tpl.displayName || tpl.name}</span>
            {#if (tpl.missingFonts?.length ?? 0) > 0}
              <span class="option-warn" title="缺少字体"><AlertTriangle size={10} /></span>
            {/if}
          </button>
        {/each}
        {#if filtered.length === 0}
          {#if templates.length === 0}
            <div class="selector-empty-guide">
              <ShoppingBag size={16} />
              <span>尚未安装模板</span>
              <button class="guide-link" onclick={() => { open = false; goto('/store-templates'); }}>前往模板商店</button>
            </div>
          {:else}
            <div class="selector-empty">无匹配模板</div>
          {/if}
        {/if}
      </div>
    </div>
  {/if}
</div>

<style>
  .selector-wrapper {
    position: relative;
    -webkit-app-region: no-drag;
  }
  .selector-trigger {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    padding: 5px 10px;
    background: var(--color-surface);
    color: var(--color-text);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-sm);
    font-size: 12px;
    font-family: var(--font-ui);
    min-width: 140px;
    cursor: pointer;
    transition: border-color var(--transition);
  }
  .selector-trigger:hover {
    border-color: var(--color-muted);
  }
  .selector-trigger.open {
    border-color: var(--color-accent);
  }
  .selector-trigger.loading {
    opacity: 0.7;
    cursor: wait;
  }
  .selector-trigger :global(svg) {
    transition: transform 150ms ease;
    flex-shrink: 0;
  }
  .selector-trigger.open :global(svg) {
    transform: rotate(180deg);
  }
  .loading-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: var(--color-accent);
    animation: pulse 1s ease-in-out infinite;
  }
  @keyframes pulse {
    0%, 100% {
      opacity: 0.4;
    }
    50% {
      opacity: 1;
    }
  }
  .selector-label {
    flex: 1;
    text-align: left;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .selector-dropdown {
    position: absolute;
    top: calc(100% + 4px);
    left: 0;
    min-width: 100%;
    max-height: 280px;
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    box-shadow: var(--shadow-md);
    z-index: 100;
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }
  .selector-search {
    display: flex;
    align-items: center;
    gap: var(--space-xs);
    padding: var(--space-sm);
    border-bottom: 1px solid var(--color-border);
    color: var(--color-muted);
    flex-shrink: 0;
  }
  .selector-search input {
    flex: 1;
    background: none;
    border: none;
    color: var(--color-text);
    font-size: 12px;
    font-family: var(--font-ui);
    outline: none;
  }
  .selector-search input::placeholder { color: var(--color-muted); }
  .selector-options {
    overflow-y: auto;
    max-height: 240px;
    padding: var(--space-xs) 0;
  }
  .selector-option {
    display: flex;
    align-items: center;
    width: 100%;
    text-align: left;
    padding: 6px 12px;
    background: none;
    border: none;
    color: var(--color-text);
    font-size: 12px;
    font-family: var(--font-ui);
    cursor: pointer;
    transition: background var(--transition);
  }
  .option-label {
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .option-warn {
    color: var(--color-warning);
    flex-shrink: 0;
    display: flex;
    align-items: center;
    margin-left: 6px;
  }
  .selector-warn {
    color: var(--color-warning);
    flex-shrink: 0;
    display: flex;
    align-items: center;
  }
  .selector-option:hover,
  .selector-option.highlighted {
    background: var(--color-surface-hover);
  }
  .selector-option.selected {
    color: var(--color-accent);
    font-weight: 500;
  }
  .selector-empty {
    padding: var(--space-md);
    text-align: center;
    color: var(--color-muted);
    font-size: 12px;
  }
  .selector-empty-guide {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--space-sm);
    padding: var(--space-lg) var(--space-md);
    color: var(--color-muted);
    font-size: 12px;
  }
  .guide-link {
    background: none;
    border: none;
    color: var(--color-accent);
    font-size: 12px;
    font-family: var(--font-ui);
    cursor: pointer;
    text-decoration: underline;
    text-underline-offset: 2px;
  }
  .guide-link:hover {
    opacity: 0.8;
  }
</style>
