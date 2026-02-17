<script lang="ts">
  let { previewUrl = '' }: { previewUrl?: string } = $props();
</script>

<div class="preview-container" role="region" aria-label="PDF 预览">
  {#if previewUrl}
    <object data={previewUrl} type="application/pdf" class="pdf-viewer" aria-label="PDF 文档预览">
      <p>无法显示 PDF 预览</p>
    </object>
  {:else}
    <div class="placeholder">
      <svg xmlns="http://www.w3.org/2000/svg" width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z"/><polyline points="14 2 14 8 20 8"/></svg>
      <p>在左侧编辑 Markdown，选择模板后预览将在此显示</p>
    </div>
  {/if}
</div>

<style>
  .preview-container {
    height: 100%;
    overflow: hidden;
    background: var(--color-primary);
    border-left: 1px solid var(--color-border);
  }
  .pdf-viewer {
    width: 100%;
    height: 100%;
    border: none;
  }
  .placeholder {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    height: 100%;
    color: var(--color-muted);
    gap: var(--space-md);
    padding: var(--space-lg);
  }
  .placeholder p {
    font-size: 0.875rem;
    margin: 0;
  }
</style>
