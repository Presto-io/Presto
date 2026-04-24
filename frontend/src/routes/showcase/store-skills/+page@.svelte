<script lang="ts">
  import { onMount } from 'svelte';
  import SkillStoreView from '$lib/components/SkillStoreView.svelte';

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

<SkillStoreView
  mode="web"
  registryUrl="https://presto.c-1o.top/agent-skills/registry.json"
  title="Agent 技能"
  readmeUrl={(skill) => `https://raw.githubusercontent.com/${skill.repo}/main/${skill.path}/SKILL.md`}
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
