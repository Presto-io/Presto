<script lang="ts">
  import { firstLaunchStore } from '$lib/stores/first-launch.svelte';

  let { state: launchState, getManualDownloadUrl } = firstLaunchStore;
</script>

{#if launchState.isActive || launchState.errorMessage}
  <div class="status-text">
    {#if launchState.isActive}
      <span>正在下载默认模板... {launchState.downloaded + launchState.failed}/{launchState.total}</span>
    {:else if launchState.errorMessage}
      <span class="error">{launchState.errorMessage}</span>
      {#if launchState.failed > 0}
        {#each [...launchState.templates.values()] as tpl (tpl.name)}
          {#if tpl.status === 'error'}
            {@const manualDownloadUrl = getManualDownloadUrl(tpl.name)}
            {#if manualDownloadUrl}
              <a
                href={manualDownloadUrl}
                target="_blank"
                rel="noopener noreferrer"
              >
                下载 {tpl.name}
              </a>
            {/if}
          {/if}
        {/each}
      {/if}
    {/if}
  </div>
{/if}

<style>
  .status-text {
    padding: 3px 12px;
    font-size: 12px;
    color: var(--color-muted, #999);
    background: var(--color-surface, #1e1e1e);
    border-top: 1px solid var(--color-border, #333);
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .error {
    color: var(--color-danger, #ef4444);
  }

  a {
    color: var(--color-accent, #3b82f6);
    text-decoration: underline;
    font-size: 12px;
  }
</style>
