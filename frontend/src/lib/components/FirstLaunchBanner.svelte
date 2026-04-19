<script lang="ts">
  import { firstLaunchStore } from '$lib/stores/first-launch.svelte';
  import { fly } from 'svelte/transition';

  let { state: launchState, getManualDownloadUrl } = firstLaunchStore;
</script>

{#if launchState.isActive || launchState.errorMessage}
  <div class="banner" transition:fly={{ y: 30, duration: 300 }}>
    {#if launchState.isActive}
      <div class="progress-info">
        <span class="text">
          正在下载默认模板... {launchState.downloaded + launchState.failed}/{launchState.total}
        </span>
        <div class="progress-bar">
          <div
            class="progress-fill"
            style="width: {((launchState.downloaded + launchState.failed) / launchState.total) * 100}%"
          ></div>
        </div>
      </div>
    {:else if launchState.errorMessage}
      <div class="error-info">
        <span class="error-text">{launchState.errorMessage}</span>
        {#if launchState.failed > 0}
          <details class="manual-download">
            <summary>手动下载模板</summary>
            <div class="download-links">
              {#each [...launchState.templates.values()] as tpl (tpl.name)}
                {#if tpl.status === 'error'}
                  <a
                    href={getManualDownloadUrl(tpl.name)}
                    target="_blank"
                    rel="noopener noreferrer"
                    class="download-link"
                  >
                    下载 {tpl.name} 模板 (zip)
                  </a>
                {/if}
              {/each}
            </div>
          </details>
        {/if}
      </div>
    {/if}
  </div>
{/if}

<style>
  .banner {
    position: fixed;
    bottom: 0;
    left: 0;
    right: 0;
    background: var(--color-surface, #1e1e1e);
    color: var(--color-muted, #999);
    padding: 4px 12px;
    z-index: 9000;
    border-top: 1px solid var(--color-border, #333);
    font-size: 12px;
  }

  .progress-info {
    display: flex;
    align-items: center;
    gap: 16px;
  }

  .text {
    font-size: 12px;
  }

  .progress-bar {
    flex: 1;
    max-width: 200px;
    height: 4px;
    background: var(--color-border, #444);
    border-radius: 2px;
    overflow: hidden;
  }

  .progress-fill {
    height: 100%;
    background: var(--color-accent, #3b82f6);
    transition: width 0.3s ease;
  }

  .error-info {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .error-text {
    font-size: 12px;
  }

  .manual-download {
    font-size: 13px;
  }

  .manual-download summary {
    cursor: pointer;
    text-decoration: underline;
  }

  .download-links {
    margin-top: 8px;
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .download-link {
    color: white;
    text-decoration: underline;
  }

  .download-link:hover {
    opacity: 0.8;
  }
</style>
