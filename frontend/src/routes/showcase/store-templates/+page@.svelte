<script lang="ts">
  import StoreView from '$lib/components/StoreView.svelte';
</script>

<StoreView
  mode="web"
  registryUrl="https://presto.c-1o.top/templates/registry.json"
  title="模板商店"
  previewUrl={(name) => `/showcase/editor?registry=${name}`}
  readmeUrl={(name) => `https://presto.c-1o.top/templates/${name}/README.md`}
/>

<style>
  :global(.app), :global(#main-content) {
    overflow: auto !important;
  }
</style>
