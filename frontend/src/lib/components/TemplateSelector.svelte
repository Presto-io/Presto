<script lang="ts">
  import { onMount } from 'svelte';
  import { listTemplates } from '$lib/api/client';
  import type { Template } from '$lib/api/types';

  let { selected = $bindable('') }: { selected?: string } = $props();
  let templates: Template[] = $state([]);

  onMount(async () => {
    const t = await listTemplates();
    templates = t ?? [];
    if (!selected && templates.length > 0) {
      selected = templates[0].name;
    }
  });
</script>

<select bind:value={selected} aria-label="选择模板" class="template-select">
  {#each templates as tpl (tpl.name)}
    <option value={tpl.name}>{tpl.displayName || tpl.name}</option>
  {/each}
</select>

<style>
  .template-select {
    padding: var(--space-sm) var(--space-md);
    background: var(--color-surface);
    color: var(--color-text);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    font-size: 0.875rem;
    min-width: 160px;
    transition: border-color var(--transition);
  }
  .template-select:hover {
    border-color: var(--color-secondary);
  }
  .template-select:focus-visible {
    outline: 2px solid var(--color-cta);
    outline-offset: 2px;
  }
</style>
