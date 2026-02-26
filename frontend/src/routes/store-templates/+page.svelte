<script lang="ts">
  import { onMount } from 'svelte';
  import StoreView from '$lib/components/StoreView.svelte';
  import { templateStore } from '$lib/stores/templates.svelte';
  import { installFromRegistry } from '$lib/api/client';
  import type { RegistryItem } from '$lib/api/types';

  let installedNames = $derived(new Set(templateStore.templates.map(t => t.name)));
  let communityEnabled = $state(false);

  async function handleInstall(tpl: RegistryItem) {
    await installFromRegistry(tpl);
    await templateStore.refresh();
  }

  onMount(() => {
    templateStore.load();
    communityEnabled = localStorage.getItem('communityTemplates') === 'true';
  });
</script>

<StoreView
  mode="desktop"
  registryUrl="https://presto.c-1o.top/templates/registry.json"
  mockRegistryUrl="/mock/registry.json"
  title="模板商店"
  installFn={handleInstall}
  {installedNames}
  previewUrl={(name) => `/showcase/editor?registry=${name}`}
  readmeUrl={(name) => `https://presto.c-1o.top/templates/${name}/README.md`}
  backRoute="/settings"
  {communityEnabled}
  statsUrl="https://registry.presto.app/api/stats"
  onInstallSuccess={async (name) => {
    try {
      await fetch(`https://registry.presto.app/api/stats/${encodeURIComponent(name)}/download`, { method: 'POST' });
    } catch {}
  }}
/>
