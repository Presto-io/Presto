<script lang="ts">
  import { onMount } from 'svelte';
  import StoreView from '$lib/components/StoreView.svelte';
  import { templateStore } from '$lib/stores/templates.svelte';
  import { installFromRegistry } from '$lib/api/client';
  import type { RegistryItem } from '$lib/api/types';
  import { page } from '$app/stores';

  declare global {
    interface Window {
      go?: { main: { App: {
        InstallTemplate: (templateName: string) => Promise<void>;
      } } };
    }
  }

  const isDev = import.meta.env.DEV || import.meta.env.VITE_MOCK === '1';
  let installedNames = $derived(new Set(templateStore.templates.map(t => t.name)));
  let communityEnabled = $state(false);

  // Read ?template= query param for deep linking (from presto:// URL scheme)
  let initialTemplate = $derived($page.url.searchParams.get('template'));

  async function handleInstall(tpl: RegistryItem) {
    // Use Wails binding to bypass WebView HTTP limitations (%2F decoding, header stripping)
    if (window.go?.main?.App?.InstallTemplate) {
      await window.go.main.App.InstallTemplate(tpl.name);
    } else {
      // Fallback for dev/web mode
      await installFromRegistry(tpl);
    }
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
  initialSelectedId={initialTemplate}
  statsUrl={isDev ? '/mock/stats.json' : 'https://registry.presto.app/api/stats'}
  onInstallSuccess={async (name) => {
    if (!isDev) {
      try {
        await fetch(`https://registry.presto.app/api/stats/${encodeURIComponent(name)}/download`, { method: 'POST' });
      } catch {}
    }
  }}
/>
