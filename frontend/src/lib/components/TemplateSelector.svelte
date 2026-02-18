<script lang="ts">
  import { onMount } from 'svelte';
  import { listTemplates } from '$lib/api/client';
  import type { Template } from '$lib/api/types';

  let {
    selected = $bindable(''),
    onbeforechange
  }: {
    selected?: string;
    onbeforechange?: (newValue: string) => void;
  } = $props();

  let templates: Template[] = $state([]);

  onMount(async () => {
    const t = await listTemplates();
    templates = t ?? [];
    if (!selected && templates.length > 0) {
      if (onbeforechange) {
        onbeforechange(templates[0].name);
      } else {
        selected = templates[0].name;
      }
    }
  });

  function handleChange(e: Event) {
    const value = (e.currentTarget as HTMLSelectElement).value;
    if (onbeforechange) {
      onbeforechange(value);
    } else {
      selected = value;
    }
  }
</script>

<select value={selected} onchange={handleChange} aria-label="选择模板" class="template-select">
  {#each templates as tpl (tpl.name)}
    <option value={tpl.name}>{tpl.displayName || tpl.name}</option>
  {/each}
</select>

<style>
  .template-select {
    padding: 5px 10px;
    background: var(--color-surface);
    color: var(--color-text);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-sm);
    font-size: 12px;
    font-family: var(--font-ui);
    min-width: 140px;
    transition: border-color var(--transition);
    -webkit-app-region: no-drag;
  }
  .template-select:hover {
    border-color: var(--color-muted);
  }
  .template-select:focus-visible {
    outline: 2px solid var(--color-accent);
    outline-offset: 2px;
  }
</style>
