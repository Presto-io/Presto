<script lang="ts">
  import { templateStore } from '$lib/stores/templates.svelte';
  import type { Template } from '$lib/api/types';
  import { onMount } from 'svelte';

  let installedTemplates = $state<string[]>([]);
  let loading = $state(true);

  onMount(async () => {
    try {
      await templateStore.load();
      installedTemplates = templateStore.templates.map((t: Template) => t.name);
    } catch (err) {
      console.error('[offline] failed to load templates:', err);
    } finally {
      loading = false;
    }
  });
</script>

<div class="offline-container">
  <div class="offline-header">
    <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      <path d="M18.36 6.64a9 9 0 1 1-12.73 0" />
      <line x1="12" y1="2" x2="12" y2="12" />
    </svg>
    <h2>离线模式</h2>
    <p class="subtitle">无网络连接,以下模板可用:</p>
  </div>

  {#if loading}
    <div class="loading">加载中...</div>
  {:else if installedTemplates.length === 0}
    <div class="empty">
      <p>未安装任何模板</p>
      <p class="hint">请连接网络后下载模板</p>
    </div>
  {:else}
    <ul class="template-list">
      {#each installedTemplates as name (name)}
        <li class="template-item">
          <span class="name">{name}</span>
        </li>
      {/each}
    </ul>
  {/if}
</div>

<style>
  .offline-container {
    padding: 48px 24px;
    max-width: 600px;
    margin: 0 auto;
  }

  .offline-header {
    text-align: center;
    margin-bottom: 32px;
  }

  .icon {
    width: 64px;
    height: 64px;
    color: #6b7280;
    margin-bottom: 16px;
  }

  h2 {
    font-size: 24px;
    font-weight: 600;
    margin: 0 0 8px 0;
    color: #1f2937;
  }

  .subtitle {
    font-size: 14px;
    color: #6b7280;
    margin: 0;
  }

  .loading {
    text-align: center;
    padding: 24px;
    color: #6b7280;
  }

  .empty {
    text-align: center;
    padding: 24px;
    color: #6b7280;
  }

  .hint {
    font-size: 13px;
    margin-top: 8px;
  }

  .template-list {
    list-style: none;
    padding: 0;
    margin: 0;
    display: grid;
    gap: 12px;
  }

  .template-item {
    display: flex;
    align-items: center;
    padding: 12px 16px;
    background: #f9fafb;
    border-radius: 8px;
    border: 1px solid #e5e7eb;
  }

  .name {
    font-weight: 500;
    color: #1f2937;
  }
</style>
