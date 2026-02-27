<script lang="ts">
  import { onMount } from 'svelte';
  import StoreView from '$lib/components/StoreView.svelte';

  onMount(() => {
    if (window.parent === window) return;
    const ro = new ResizeObserver(() => {
      window.parent.postMessage(
        { type: 'presto-resize', height: document.documentElement.scrollHeight },
        '*'
      );
    });
    ro.observe(document.documentElement);
    return () => ro.disconnect();
  });
</script>

<StoreView
  mode="web"
  registryUrl="https://presto.c-1o.top/templates/registry.json"
  title="模板商店"
  previewUrl={(name) => `/showcase/editor?registry=${name}`}
  readmeUrl={(name) => `https://presto.c-1o.top/templates/${name}/README.md`}
  statsUrl="https://registry.presto.app/api/stats"
/>

<style>
  :global(html), :global(body) {
    max-width: 100vw !important;
    overflow-x: hidden !important;
  }
  :global(body) {
    overflow-y: auto !important;
  }
  :global(.app), :global(#main-content) {
    height: auto !important;
  }
</style>
