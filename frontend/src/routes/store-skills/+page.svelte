<script lang="ts">
  import { onMount } from 'svelte';
  import SkillStoreView from '$lib/components/SkillStoreView.svelte';
  import { loadCapabilities, type ReleaseCapabilities } from '$lib/config/channel';

  let capabilities = $state<ReleaseCapabilities | null>(null);
  let capabilitiesLoaded = $state(false);

  onMount(async () => {
    capabilities = await loadCapabilities();
    capabilitiesLoaded = true;
  });
</script>

{#if !capabilitiesLoaded}
  <main class="store-disabled">
    <h2>技能商店</h2>
  </main>
{:else if capabilities && !capabilities.onlineSkillStore}
  <main class="store-disabled">
    <h2>技能商店</h2>
    <p>离线便携包已关闭在线技能商店。本地已安装技能仍可继续使用。</p>
  </main>
{:else if capabilities}
  <SkillStoreView
    mode="desktop"
    registryUrl="https://presto.c-1o.top/agent-skills/registry.json"
    title="技能商店"
    readmeUrl={(skill) => `https://raw.githubusercontent.com/${skill.repo}/main/${skill.path}/SKILL.md`}
    backRoute="/settings"
  />
{/if}

<style>
  .store-disabled {
    display: flex;
    height: 100%;
    flex-direction: column;
    justify-content: center;
    gap: var(--space-sm);
    padding: var(--space-xl);
    color: var(--color-text);
  }
  .store-disabled h2 {
    margin: 0;
    font-size: 1rem;
  }
  .store-disabled p {
    max-width: 420px;
    margin: 0;
    color: var(--color-muted);
    font-size: 0.875rem;
    line-height: 1.6;
  }
</style>
