<script lang="ts">
  import { installState, type ActiveDownloadEntry } from '$lib/stores/install-state.svelte';
  import { fly } from 'svelte/transition';
  import { onMount } from 'svelte';

  let isMac = $state(false);
  onMount(async () => {
    if (window.go?.main?.App?.GetPlatform) {
      isMac = (await window.go.main.App.GetPlatform()) === 'darwin';
    }
  });

  let downloadingTemplates = $derived.by((): ActiveDownloadEntry[] => {
    return installState.getActiveDownloads();
  });

  // Color palette (5 colors based on template name hash)
  const COLORS = ['#22c55e', '#3b82f6', '#a855f7', '#f97316', '#ef4444'];

  function getColorForTemplate(name: string): string {
    // Simple hash function
    let hash = 0;
    for (let i = 0; i < name.length; i++) {
      hash = ((hash << 5) - hash) + name.charCodeAt(i);
      hash = hash & hash;
    }
    return COLORS[Math.abs(hash) % COLORS.length];
  }

  function formatBytes(bytes: number): string {
    if (bytes < 1024) return bytes + ' B';
    if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB';
    return (bytes / (1024 * 1024)).toFixed(1) + ' MB';
  }
</script>

<div class="progress-container" class:mac-offset={isMac}>
  {#each downloadingTemplates as item (item.name)}
    <div
      class="progress-bar"
      style="--color: {getColorForTemplate(item.name)}"
      in:fly={{ y: -20, duration: 300 }}
      out:fly={{ y: -20, duration: 500 }}
    >
      <div
        class="progress-fill"
        style="width: {item.progress.percent}%"
      ></div>
      <div class="tooltip">
        <span class="name">{item.name}</span>
        <span class="size">
          {formatBytes(item.progress.downloaded)} / {formatBytes(item.progress.total)}
        </span>
      </div>
    </div>
  {/each}
</div>

<style>
  .progress-container {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    z-index: 9001;
    display: flex;
    flex-direction: column;
    gap: 0;
  }

  .progress-container.mac-offset {
    padding-left: 78px;
  }

  .progress-bar {
    position: relative;
    height: 4px;
    background: rgba(0, 0, 0, 0.1);
    overflow: visible;
  }

  .progress-fill {
    height: 100%;
    background: var(--color);
    background-image: linear-gradient(
      45deg,
      rgba(255, 255, 255, 0.15) 25%,
      transparent 25%,
      transparent 50%,
      rgba(255, 255, 255, 0.15) 50%,
      rgba(255, 255, 255, 0.15) 75%,
      transparent 75%,
      transparent
    );
    background-size: 1rem 1rem;
    animation: stripes 1s linear infinite;
    transition: width 300ms linear;
    box-shadow: 0 0 8px var(--color);
  }

  @keyframes stripes {
    from { background-position: 1rem 0; }
    to { background-position: 0 0; }
  }

  .tooltip {
    position: absolute;
    top: 100%;
    left: 50%;
    transform: translateX(-50%);
    padding: 4px 8px;
    background: rgba(0, 0, 0, 0.8);
    color: white;
    font-size: 12px;
    border-radius: 4px;
    white-space: nowrap;
    opacity: 0;
    pointer-events: none;
    transition: opacity 200ms;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .progress-bar:hover .tooltip {
    opacity: 1;
  }

  .name {
    font-weight: 500;
  }

  .size {
    opacity: 0.8;
  }
</style>
