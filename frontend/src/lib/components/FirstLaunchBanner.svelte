<script lang="ts">
  import { firstLaunchStore } from '$lib/stores/first-launch.svelte';

  let { state, getManualDownloadUrl } = firstLaunchStore;
</script>

{#if state.isActive || state.errorMessage}
  <div class="banner">
    {#if state.isActive}
      <div class="progress-info">
        <span class="text">
          正在下载默认模板... {state.downloaded + state.failed}/{state.total}
        </span>
        <div class="progress-bar">
          <div
            class="progress-fill"
            style="width: {((state.downloaded + state.failed) / state.total) * 100}%"
          ></div>
        </div>
      </div>
    {:else if state.errorMessage}
      <div class="error-info">
        <span class="error-text">{state.errorMessage}</span>
        {#if state.failed > 0}
          <details class="manual-download">
            <summary>手动下载模板</summary>
            <div class="download-links">
              {#each [...state.templates.values()] as tpl (tpl.name)}
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
    top: 0;
    left: 0;
    right: 0;
    background: #3b82f6;
    color: white;
    padding: 8px 16px;
    z-index: 100;
    box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
  }

  .progress-info {
    display: flex;
    align-items: center;
    gap: 16px;
  }

  .text {
    font-size: 14px;
  }

  .progress-bar {
    flex: 1;
    max-width: 200px;
    height: 6px;
    background: rgba(255, 255, 255, 0.3);
    border-radius: 3px;
    overflow: hidden;
  }

  .progress-fill {
    height: 100%;
    background: white;
    transition: width 0.3s ease;
  }

  .error-info {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .error-text {
    font-size: 14px;
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
